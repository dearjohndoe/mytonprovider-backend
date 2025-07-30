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

Для начала нам потребуется чистый сервер на Debian 12 с рут пользователем.

1. **Склонируйте скрипт для подключения по ключу**

Вместо логина по паролю, скрипт безопасности требует использовать логин по ключу. Этот скрипт нужно запускать на рабочей машине, он не потребует sudo, а только пробросит ключи для доступа.

```bash
wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/refs/heads/master/scripts/init_server_connection.sh
```

2. **Пробрасываем ключи и закрываем доступ по паролю**

```bash
USERNAME=root PASSWORD=supersecretpassword HOST=123.45.67.89 bash init_server_connection.sh
```

В случае ошибки man-in-the-middle, возможно вам стоит удалить known_hosts.

3. **Заходим на удаленную машину и качаем скрипт установки**

```bash
ssh root@123.45.67.89 # Если требует пароль, то предыдущий шаг завершился с ошибкой.

wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/refs/heads/master/scripts/setup_server.sh
```

4. **Запускаем настройку и установку сервера**

Займет несколько минут.

```bash
PG_USER=pguser PG_PASSWORD=secret PG_DB=providerdb NEWFRONTENDUSER=jdfront NEWSUDOUSER=johndoe NEWUSER_PASSWORD=newsecurepassword bash ./setup_server.sh
```

По завершении выведет полезную информацию по использованию сервера.


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



Этот проект был создан по заказу участника сообщества TON Foundation.
Оплата была произведена по адресу:
UQB0T1-iAtlArjW6feQb7SVuZFiDc_JjhqwWRFrzzh6yS0Q8
