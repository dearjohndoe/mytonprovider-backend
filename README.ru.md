# mytonprovider-backend

Backend сервис для mytonprovider.org - сервис мониторинга провайдеров TON Storage.

## Описание

Данный backend сервис:
- Взаимодействует с провайдерами хранилища через ADNL протокол
- Мониторит производительность, доступность провайдеров, доступность хранимых файлов, проводит проверки здоровья
- Обрабатывает телеметрию от провайдеров
- Предоставляет API эндпоинты для фронтенда
- Вычисляет рейтинг, аптайм, статус провайдеров
- Собирает собственные метрики через **Prometheus**

## Установка и настройка

1. **Склонируйте репозиторий**
   ```bash
   git clone https://github.com/dearjohndoe/mytonprovider-backend.git
   cd ton-provider-org
   ```

2. **Запуск скрипта установки**
**DOMAIN** и **INSTALL_SSL** не обязательны.
Этот скрипт должен быть запущен на чистом сервере с рут пользователя (был протестирован на чистом Debian 12 с рутом)

```bash
REMOTEUSER=root \
HOST=123.45.67.89 \
PASSWORD=yourpassword \
PG_VERSION=15 \
PG_USER=pguser \
PG_PASSWORD=secret \
PG_DB=providerdb \
NEWSUDOUSER=johndoe \
NEWUSER_PASSWORD=newsecurepassword \
DOMAIN=domain_u_own.org \
INSTALL_SSL=true \
./setup_server.sh
```

## Разработка

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
            "buildFlags": "-tags=debug",    // для обработки OPTIONS запросов без nginx при разработке
            "env": {...}
        }
    ]
}
```

## Структура проекта

```
├── cmd/                   # Точка входа приложения, конфиги, инициализация
├── pkg/                   # Пакеты приложения
│   ├── cache/             # Кастомный кеш
│   ├── httpServer/        # Fiber хандлеры сервера
│   ├── models/            # Модели данных для БД и API
│   ├── repositories/      # Вся работа с postgres здесь
│   ├── services/          # Бизнес логика
│   ├── tonclient/         # TON blockchain клиент, обертка для нескольких полезных функций
│   └── workers/           # Воркеры
├── db/                    # Схема базы данных
├── scripts/               # Скрипты настройки и утилиты
```

## API Эндпоинты

Сервер предоставляет REST API эндпоинты для:
- Сбора телеметрии провайдеров
- Информации о провайдерах и инструменты фильтрации
- Метрик

## Воркеры

Приложение запускает несколько фоновых воркеров:
- **Providers Master**: Управляет жизненным циклом провайдеров, проверками здоровья и хранимых файлов
- **Telemetry Worker**: Обрабатывает входящюю телеметрию
- **Cleaner Worker**: Чистит базу данных от устаревшей информации

## Лицензия

Apache-2.0
