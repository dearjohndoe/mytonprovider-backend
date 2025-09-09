# mytonprovider-backend

**[Русская версия](README.ru.md)**

Backend service for mytonprovider.org - a TON Storage providers monitoring service.

## Description

This backend service:
- Communicates with storage providers via ADNL protocol
- Monitors provider performance, availability, do health checks
- Handles telemetry data from providers
- Provides API endpoints for frontend
- Computes provider ratings
- Collect own metrics via **Prometheus**

## Installation & Setup

To get started, you'll need a clean Debian 12 server with root user access.

1. **Download the server connection script**

Instead of password login, the security script requires using key-based authentication. This script should be run on your local machine, it doesn't require sudo, and will only forward keys for access.

```bash
wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/refs/heads/master/scripts/init_server_connection.sh
```

2. **Forward keys and disable password access**

```bash
USERNAME=root PASSWORD=supersecretpassword HOST=123.45.67.89 bash init_server_connection.sh
```

In case of a man-in-the-middle error, you might need to remove known_hosts.

3. **Log into the remote machine and download the installation script**

```bash
ssh root@123.45.67.89 # If it asks for a password, the previous step failed.

wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/refs/heads/master/scripts/setup_server.sh
```

4. **Run server setup and installation**

This will take a few minutes.

```bash
PG_USER=pguser PG_PASSWORD=secret PG_DB=providerdb NEWFRONTENDUSER=jdfront NEWSUDOUSER=johndoe NEWUSER_PASSWORD=newsecurepassword bash ./setup_server.sh
```

Upon completion, it will output useful information about server usage.

## Dev:
### VS Code Configuration
Create `.vscode/launch.json`:
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
            "buildFlags": "-tags=debug",    // to handle OPTIONS queries without nginx when dev
            "env": {...}
        }
    ]
}
```

## Project Structure

```
├── cmd/                   # Application entry point, configs, inits
├── pkg/                   # Application packages
│   ├── cache/             # Custom cache
│   ├── httpServer/        # Fiber server handlers
│   ├── models/            # DB and API data models
│   ├── repositories/      # All work with postgres here
│   ├── services/          # Business logic
│   ├── tonclient/         # TON blockchain client, wrap some usefull functions
│   └── workers/           # Workers
├── db/                    # Database schema
├── scripts/               # Setup and utility scripts
```

## API Endpoints

The server provides REST API endpoints for:
- Telemetry data collection
- Provider info and filters tool
- Metrics

## Workers

The application runs several background workers:
- **Providers Master**: Manages provider lifecycle and health checks
- **Telemetry Worker**: Processes incoming telemetry data
- **Cleaner Worker**: Maintains database hygiene and cleanup

## License
 
Apache-2.0



This project was created by order of a TON Foundation community member.
