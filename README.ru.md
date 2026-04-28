# mytonprovider-backend

Backend сервис для mytonprovider.org — сервис мониторинга провайдеров TON Storage.

## Описание

Данный backend сервис:
- Взаимодействует с провайдерами хранилища через ADNL протокол
- Мониторит производительность, доступность провайдеров и проводит проверки здоровья
- Обрабатывает телеметрию от провайдеров
- Предоставляет REST API эндпоинты для фронтенда
- Вычисляет рейтинг провайдеров
- Собирает собственные метрики через **Prometheus**

## Установка и настройка

Для начала потребуется чистый сервер на Debian 12 с доступом от root пользователя.

1. **Подключитесь к серверу и скачайте скрипт установки**

```bash
ssh root@123.45.67.89

wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/master/scripts/setup_server.sh
chmod +x setup_server.sh
```

2. **Запустите установку сервера**

Займёт несколько минут.

```bash
DB_HOST=db DB_USER=pguser DB_PASSWORD=secret DB_NAME=providerdb \
MASTER_ADDRESS=UQD...адрес_мастер_контракта... \
NEWSUDOUSER=johndoe NEWUSER_PASSWORD=newsecurepassword \
NEWFRONTENDUSER=jdfront \
DOMAIN=mytonprovider.org INSTALL_SSL=true \
bash ./setup_server.sh
```

Скрипт выполнит:
- Установку Docker и системных зависимостей
- Клонирование репозитория в `/opt/provider`
- Создание `.env` и запуск стека Docker Compose
- Настройку Nginx reverse proxy
- Защиту сервера (UFW, fail2ban, вход только по ключу SSH, отключение root логина)
- Сборку и деплой фронтенда

По завершении выведет полезные команды для управления сервером.

**Обязательные переменные:** `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `MASTER_ADDRESS`, `NEWSUDOUSER`, `NEWUSER_PASSWORD`, `NEWFRONTENDUSER`

**Опциональные переменные:** `DOMAIN` (по умолчанию — IP сервера), `INSTALL_SSL` (`true`/`false`), `DB_PORT` (по умолчанию `5432`), `SYSTEM_PORT` (по умолчанию `9090`)

> `DB_HOST` должен быть `db`, когда запускается стандартный стек Docker Compose (это имя сервиса в `docker-compose.yml`). Используйте `localhost` или внешний хост только если приложение работает вне Docker.
>
> `MASTER_ADDRESS` — адрес мастер-контракта TON, с которого бэкенд сканирует транзакции. Принимается в user-friendly формате (`UQ...`/`EQ...`) или raw (`0:abc...`).

## Разработка

### Локальная установка

Требуется: **Docker** (с плагином compose) и **Go 1.24+**.

```bash
bash scripts/setup_local.sh
```

Скрипт выполнит:
- Создание `.env` из `.env.example` (если `.env` не существует)
- Сборку Docker образа
- Запуск всех сервисов: PostgreSQL 15, миграции базы данных и бэкенд

Просмотр логов:
```bash
docker compose -f docker-compose.yml logs -f app
```

Пересборка после изменений кода:
```bash
docker compose -f docker-compose.yml up -d --build app
```

Остановка всех сервисов:
```bash
docker compose -f docker-compose.yml down
```

### Переменные окружения

Скопируйте `.env.example` в `.env` и настройте значения:

| Переменная | По умолчанию | Описание |
|---|---|---|
| `MASTER_ADDRESS` | — | Адрес мастер-контракта TON |
| `SYSTEM_ACCESS_TOKENS` | — | MD5-хэши валидных токенов через запятую |
| `SYSTEM_PORT` | `9090` | Порт HTTP сервера |
| `DB_HOST` | `db` | Хост PostgreSQL (`db` для Docker, `localhost` для bare metal) |
| `DB_PORT` | `5432` | Порт PostgreSQL |
| `DB_USER` | — | Пользователь PostgreSQL |
| `DB_PASSWORD` | — | Пароль PostgreSQL |
| `DB_NAME` | — | Имя базы данных PostgreSQL |
| `SYSTEM_LOG_LEVEL` | `1` | Уровень логов: 0=debug, 1=info, 2=warn, 3=error |
| `CONFIG_PATH` | — | Путь к YAML конфигу (например `config/dev.yaml`) |

### Конфигурация VS Code

Создайте `.vscode/launch.json`:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd",
            "buildFlags": "-tags=debug",
            "envFile": "${workspaceFolder}/.env"
        }
    ]
}
```

## Структура проекта

```
├── cmd/                   # Точка входа приложения и инициализация
├── config/                # YAML конфиги (например dev.yaml)
├── pkg/                   # Пакеты приложения
│   ├── cache/             # Кэш в памяти
│   ├── clients/           # Клиенты внешних сервисов (TON, ifconfig)
│   ├── config/            # Загрузчик конфига (cleanenv)
│   ├── httpServer/        # Fiber HTTP сервер, хандлеры, middleware
│   ├── metrics/           # Определения метрик Prometheus
│   ├── models/            # Модели данных для БД и API
│   ├── repositories/      # Запросы к PostgreSQL
│   ├── services/          # Бизнес-логика
│   └── workers/           # Фоновые воркеры
├── scripts/               # Скрипты настройки и утилиты
├── Dockerfile             # Многоэтапная Docker сборка
├── docker-compose.yml     # Локальный / продакшн стек
└── docker-compose.test.yml # End-to-end тест setup_server.sh в контейнере
```

## Тестирование `setup_server.sh`

`docker-compose.test.yml` прогоняет полный сценарий `setup_server.sh` внутри одноразового Debian-контейнера, используя Docker-демон хоста. Так можно проверить скрипт без реального сервера.

**Из WSL** (на Windows обязательно — см. примечание ниже):

```bash
cd /mnt/c/path/to/mytonprovider-backend
docker compose -f docker-compose.test.yml up --build
```

Что происходит:
- Тестовый контейнер ставит Docker CLI, генерит SSH ключи и запускает `setup_server.sh`
- `SKIP_CLONE=true` (проект примонтирован в контейнер), `SKIP_APP_START=false`, `INSTALL_SSL=false`
- Сервисы `app`, `db`, `db_migrate` поднимаются на хостовом Docker-демоне через общий `/var/run/docker.sock`
- После setup доступ:
  - App напрямую: `http://localhost:${SYSTEM_PORT}` (по умолчанию `9090`)
  - DB напрямую: `localhost:${DB_PORT}` (по умолчанию `5432`)
  - Nginx внутри tester на хост не проброшен — тест только проверяет что он корректно установился и настроился

> **Про Windows:** запускать нужно из WSL, не из PowerShell. На Windows `${PWD}` разворачивается в `C:\...`, двоеточие ломает парсинг docker volume, плюс хостовый демон не найдёт такие пути. WSL отдаёт `/mnt/c/...` — Linux-стиль путь, который Docker Desktop корректно понимает с обеих сторон.

## API Эндпоинты

Лимит запросов: **100 запросов за 60 секунд** (скользящее окно).

| Метод | Путь | Авторизация | Описание |
|---|---|---|---|
| `GET` | `/health` | — | Проверка здоровья |
| `GET` | `/metrics` | ✓ | Метрики Prometheus |
| `POST` | `/api/v1/providers/search` | — | Поиск провайдеров с фильтрами |
| `GET` | `/api/v1/providers/filters` | — | Получить диапазоны значений для фильтров |
| `POST` | `/api/v1/providers` | — | Отправить телеметрию провайдера |
| `GET` | `/api/v1/providers` | ✓ | Получить последнюю телеметрию всех провайдеров |
| `POST` | `/api/v1/contracts/statuses` | — | Получить статусы storage контрактов |
| `POST` | `/api/v1/benchmarks` | — | Отправить данные бенчмарков |

### Авторизация

Защищённые эндпоинты (`✓`) требуют заголовок `Authorization`:

```
Authorization: Bearer <raw-token>
```

Сервер проверяет токен, вычисляя его MD5-хэш и сравнивая с `SYSTEM_ACCESS_TOKENS` в `.env`. Для добавления токена:

```bash
echo -n "your-secret-token" | md5sum
# скопируйте хэш в SYSTEM_ACCESS_TOKENS в .env
```

Несколько токенов разделяются запятой: `SYSTEM_ACCESS_TOKENS=hash1,hash2`

## Воркеры

Приложение запускает несколько фоновых воркеров:
- **Providers Master**: Управляет жизненным циклом провайдеров, проверками здоровья и доступностью хранимых файлов
- **Telemetry Worker**: Обрабатывает входящую телеметрию
- **Cleaner Worker**: Удаляет устаревшие данные из базы

## Лицензия

Apache-2.0

Этот проект был создан по заказу участника сообщества TON Foundation.
