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

1. **Clone the repository**
```bash
git clone https://github.com/dearjohndoe/mytonprovider-backend.git
cd ton-provider-org
```

2. **Run installation script**

**DOMAIN** and **INSTALL_SSL** is optional.
This script must be run only on clean server with root user (was tested only on debian 12 with root)

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
