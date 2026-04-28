# Setup: backend + agents (Redis Streams architecture)

## Архитектура в двух словах

```
┌──────────────┐       triggers (cron)        ┌──────────────┐
│              │ ───── mtpa:cycle:<type> ────▶ │              │
│   Backend    │                               │   Agent(s)   │
│              │ ◀──── mtpa:result:<type> ──── │              │
└──┬───────────┘                               └──────────────┘
   │ writes
   ▼
┌──────────────┐
│  Postgres    │
└──────────────┘
```

- **Backend** — единственный writer Postgres + cron-планировщик триггеров.
- **Agent** — stateless воркер: читает свой `mtpa:cycle:<type>`, выполняет цикл (TON liteserver, ADNL, DHT, ifconfig), публикует результат в `mtpa:result:<type>`.
- **Idempotency**: каждый job_id (UUID) попадает в `system.processed_jobs`. Повторная доставка → COMMIT пустой → дубль не применяется.
- **Single writer**: гонок на запись нет; backend применяет всё в одной транзакции с dedup.

## Что развернуть

### 1. Postgres + миграции

База с миграциями `db/000001_init.up.sql` и `db/000002_processed_jobs.up.sql`. В docker-compose это уже собрано.

```bash
cd mytonprovider-backend
cp .env.example .env  # отредактируйте DB_*, MASTER_ADDRESS
docker compose up -d db redis db_migrate
```

### 2. Backend

```bash
docker compose up -d app
docker compose logs -f app
```

Backend сам:
- подхватит расписание из `config/dev.yaml` (раздел `cycles`);
- создаст consumer-group `mtpa-backend` на каждом `mtpa:result:<type>`;
- начнёт лить триггеры по интервалам.

### 3. Agent(s)

Агент — отдельный репозиторий [`mytonprovider-agent`](../mytonprovider-agent). Строится из своего `cmd/app`. Доступ к Postgres агенту больше **не нужен**: он читает только триггеры из Redis и пишет результаты в Redis.

На машине агента:

```bash
cd mytonprovider-agent
cp .env.example .env
# Отредактируйте:
#   REDIS_ADDR=<host бэкенда>:6379   (или внутренний адрес VPN/VPC)
#   TON_CONFIG_URL=https://ton-blockchain.github.io/global.config.json
docker compose up -d
```

Агент сразу подключится к `REDIS_ADDR`, создаст consumer-group `mtpa` (по умолчанию), начнёт ждать триггеры.

## Сетевой доступ агента к Redis

Redis — единственный канал между бэкендом и агентом. Доступ нужно открыть только к 6379, не к Postgres.

### Вариант A: одна VPC / приватная сеть

Если агент и бэкенд в одной сети — просто `REDIS_ADDR=<private_ip>:6379`. Никакой конфигурации Redis не требуется.

### Вариант B: разные машины через VPN

Поднимите Tailscale / WireGuard. На бэкенде Redis слушает на интерфейсе VPN:

```yaml
# docker-compose.yml
redis:
  image: redis:7-alpine
  ports:
    - "100.64.0.1:6379:6379"   # bind на tailscale-IP, не 0.0.0.0
```

### Вариант C: публичный Redis с TLS

Не рекомендую без острой необходимости. Если всё-таки нужно:

1. Включите `requirepass` в Redis:
   ```
   redis-server --requirepass strong_password --port 6379
   ```
2. Перед Redis поставьте TLS-терминатор (stunnel / nginx stream / Envoy).
3. На агенте: `REDIS_ADDR=<host>:6380 REDIS_PASSWORD=...` + поднять stunnel-клиент.

## Проверка корректности

После старта бэкенда + хотя бы одного агента:

### 1. Стримы и группы созданы

```bash
docker compose exec redis redis-cli
> KEYS mtpa:*
1) "mtpa:cycle:probe_rates"
2) "mtpa:cycle:scan_master"
... (после первых триггеров)
1) "mtpa:result:probe_rates"
... (после первых результатов)

> XINFO GROUPS mtpa:cycle:probe_rates
1)  1) "name"
    2) "mtpa"     # consumer-group агента
2)  ...
```

### 2. Триггеры приходят

```bash
docker compose logs -f app | grep "triggered"
# {"msg":"triggered","cycle":"probe_rates","job_id":"<uuid>"}
```

### 3. Результаты применяются

```bash
docker compose logs -f app | grep "result applied"
# {"msg":"result applied","cycle":"probe_rates","job_id":"<uuid>","status":"ok"}
```

### 4. Dedup работает

```bash
docker compose exec db psql -U pguser -d providerdb \
  -c "SELECT type, count(*) FROM system.processed_jobs GROUP BY type;"
#  type              | count
# -------------------+-------
#  scan_master       |   12
#  probe_rates       |  723
#  ...
```

### 5. Тестовая нагрузка вручную

В репозитории агента есть `cmd/loadtest`:

```bash
# на машине агента или с доступом к Redis:
go run ./cmd/loadtest -addr <redis>:6379 -cycle scan_master -count 5
# → trigger #1 job_id=...
# ← ok    job_id=... agent=agent-...
```

Полезно для проверки конкретного цикла без ожидания cron-расписания.

## Конфигурация Redis-стримов

| Параметр                                    | Значение по умолчанию | Где задаётся                  |
|---------------------------------------------|-----------------------|-------------------------------|
| Префикс ключей                              | `mtpa`                | backend `redis.stream_prefix` + agent `redis.stream_prefix` |
| Consumer group (backend → reads results)    | `mtpa-backend`        | backend `redis.group`         |
| Consumer group (agent → reads triggers)     | `mtpa`                | agent `redis.group`           |
| MAXLEN result-стримов (на стороне агента)   | 100000                | agent `redis.result_maxlen`   |
| MAXLEN trigger-стримов (на стороне бэкенда) | 10000                 | backend `redis.trigger_maxlen`|

**Важно:** `stream_prefix` должен совпадать у backend и agent — иначе они будут общаться через разные ключи.

## Кастомизация расписания

В `config/dev.yaml` бэкенда секция `cycles`:

```yaml
cycles:
  probe_rates:       { enabled: true, interval: 1m,  single_inflight: true  }
  scan_master:       { enabled: true, interval: 5m,  single_inflight: true  }
  ...
```

- `interval` — пауза между триггерами одного типа.
- `single_inflight: true` — перед XADD проверять `XPENDING + Lag`. Если предыдущий триггер ещё не отработан — пропустить итерацию. Защищает от двух параллельных `scan_master` с одним `from_lt`.
- `enabled: false` — цикл выключен (ни триггеры, ни consumer не запускаются).

## Кастомизация пулов на агенте

В `config/config.yaml` агента секция `workers` (см. `config/config.example.yaml`):

```yaml
workers:
  discovery:
    enabled: true
    pool: 1               # consumer'ов на каждый цикл discovery
    timeout: 30m
    concurrency: 16       # внутренняя параллельность (resolve_endpoints)
  poll:
    enabled: true
    pool: 1
    timeout: 10m
    concurrency: 30       # параллельных probe одновременно
  ...
```

`pool` управляет тем, сколько триггеров одного типа агент может обрабатывать параллельно. `concurrency` — сколько goroutine'ов внутри одного цикла (например, probe одного триггера на N pubkey'ев).

## Multi-agent

Если агентов несколько — все запускаются с одинаковым `redis.group` (по умолчанию `mtpa`). Redis Streams сами раздают сообщения между consumers внутри группы — каждый триггер достаётся ровно одному агенту.

`AGENT_ID` должен быть уникален. По умолчанию `auto` → UUID на старте, поэтому даже если запустить N экземпляров с одинаковым конфигом — у каждого свой ID.

## Что если агент упадёт во время обработки

1. Сообщение остаётся в `XPENDING` группы `mtpa`.
2. После рестарта тот же агент возьмёт его обратно (consumer-id персистентен через `AGENT_ID`).
3. Если агент уже не вернётся — другой агент в группе сделает `XAUTOCLAIM` (этот reaper в текущей версии не реализован, можно добавить отдельным cron-job'ом):

```bash
# ручной cleanup (одноразовый)
redis-cli XAUTOCLAIM mtpa:cycle:probe_rates mtpa another-agent 600000 0
```

Бэкенд после получения результата применяет его транзакционно, поэтому переобработка не порождает гонок: dedup-таблица `system.processed_jobs` отсекает дубликаты.

## Что если бэкенд упадёт во время записи

1. Транзакция откатывается, `XACK` не делается.
2. После рестарта backend снова прочитает то же сообщение из `mtpa:result:<type>`.
3. INSERT INTO `processed_jobs` либо проходит впервые (значит транзакция реально не дошла до commit'а раньше), либо возвращает 0 строк (дубль) — оба случая корректны.
4. После успешного commit'а делается `XACK`.

## Откат на старую архитектуру

Сохраните старый коммит. Откат состоит из:
1. `git checkout <pre-redis-commit>`
2. Удалить миграцию `000002_processed_jobs.down.sql up`.
3. Понизить `go.mod` (без `redis/go-redis/v9`).

Данные в Postgres схема совместимая — таблицы `providers.*` и `system.params` не менялись, только добавилась `system.processed_jobs`.
