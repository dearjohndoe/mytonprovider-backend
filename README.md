# mytonprovider-backend

**[Русская версия](README.ru.md)**

Backend service for mytonprovider.org - a TON Storage providers monitoring service.

## Description

This backend service:
- Communicates with storage providers via ADNL protocol
- Monitors provider performance, availability, and does health checks
- Handles telemetry data from providers
- Provides REST API endpoints for frontend
- Computes provider ratings
- Collects own metrics via **Prometheus**

## Installation & Setup

To get started, you'll need a clean Debian 12 server with root user access.

1. **SSH into the server and download the setup script**

```bash
ssh root@123.45.67.89

wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/master/scripts/setup_server.sh
chmod +x setup_server.sh
```

2. **Run server setup**

This will take a few minutes.

```bash
DB_HOST=db DB_USER=pguser DB_PASSWORD=secret DB_NAME=providerdb \
MASTER_ADDRESS=UQD...your_ton_master_contract_address... \
NEWSUDOUSER=johndoe NEWUSER_PASSWORD=newsecurepassword \
NEWFRONTENDUSER=jdfront \
DOMAIN=mytonprovider.org INSTALL_SSL=true \
bash ./setup_server.sh
```

The script will:
- Install Docker and system dependencies
- Clone the repository to `/opt/provider`
- Create `.env` and start the Docker Compose stack
- Configure Nginx reverse proxy
- Harden the server (UFW, fail2ban, SSH key-only auth, disable root login)
- Build and deploy the frontend

Upon completion, it will print useful commands for managing the server.

**Required variables:** `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `MASTER_ADDRESS`, `NEWSUDOUSER`, `NEWUSER_PASSWORD`, `NEWFRONTENDUSER`

**Optional variables:** `DOMAIN` (defaults to server IP), `INSTALL_SSL` (`true`/`false`), `DB_PORT` (default `5432`), `SYSTEM_PORT` (default `9090`)

> `DB_HOST` should be `db` when running the standard Docker Compose stack (that's the service name in `docker-compose.yml`). Use `localhost` or an external hostname only if the app runs outside Docker.
>
> `MASTER_ADDRESS` is the address of the TON master smart contract the backend scans transactions for. Accepted in user-friendly form (`UQ...`/`EQ...`) or raw (`0:abc...`).

## Dev

### Local Setup

Requires: **Docker** (with compose plugin) and **Go 1.24+**.

```bash
bash scripts/setup_local.sh
```

This will:
- Create `.env` from `.env.example` (if `.env` doesn't exist)
- Build the Docker image
- Start all services: PostgreSQL 15, database migrations, and the backend app

View logs:
```bash
docker compose -f docker-compose.yml logs -f app
```

Rebuild after code changes:
```bash
docker compose -f docker-compose.yml up -d --build app
```

Stop all services:
```bash
docker compose -f docker-compose.yml down
```

### Environment Variables

Copy `.env.example` to `.env` and adjust values:

| Variable | Default | Description |
|---|---|---|
| `MASTER_ADDRESS` | — | TON master contract address |
| `SYSTEM_ACCESS_TOKENS` | — | Comma-separated MD5 hashes of valid API tokens |
| `SYSTEM_PORT` | `9090` | HTTP server port |
| `DB_HOST` | `db` | PostgreSQL host (use `db` for Docker, `localhost` for bare metal) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | — | PostgreSQL user |
| `DB_PASSWORD` | — | PostgreSQL password |
| `DB_NAME` | — | PostgreSQL database name |
| `SYSTEM_LOG_LEVEL` | `1` | Log level: 0=debug, 1=info, 2=warn, 3=error |
| `CONFIG_PATH` | — | Path to YAML config file (e.g. `config/dev.yaml`) |

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
            "buildFlags": "-tags=debug",
            "envFile": "${workspaceFolder}/.env"
        }
    ]
}
```

## Project Structure

```
├── cmd/                   # Application entry point and initialization
├── config/                # YAML config files (e.g. dev.yaml)
├── pkg/                   # Application packages
│   ├── cache/             # In-memory cache
│   ├── clients/           # External service clients (TON, ifconfig)
│   ├── config/            # Config loader (cleanenv)
│   ├── httpServer/        # Fiber HTTP server, handlers, middleware
│   ├── metrics/           # Prometheus metric definitions
│   ├── models/            # DB and API data models
│   ├── repositories/      # PostgreSQL queries
│   ├── services/          # Business logic
│   └── workers/           # Background workers
├── scripts/               # Setup and utility scripts
├── Dockerfile             # Multi-stage Docker build
├── docker-compose.yml     # Local / production stack
└── docker-compose.test.yml # End-to-end test of setup_server.sh in a container
```

## Testing `setup_server.sh`

`docker-compose.test.yml` runs the full `setup_server.sh` flow inside a throwaway Debian container against the host Docker daemon, so you can verify the script without provisioning a real server.

**From WSL** (required on Windows — see note below):

```bash
cd /mnt/c/path/to/mytonprovider-backend
docker compose -f docker-compose.test.yml up --build
```

What happens:
- The tester container installs Docker CLI, generates SSH keys, and runs `setup_server.sh`
- `SKIP_CLONE=true` (the project is mounted into the tester), `SKIP_APP_START=false`, `INSTALL_SSL=false`
- The `app`, `db`, `db_migrate` services start on the host Docker daemon via the shared `/var/run/docker.sock`
- After setup, access:
  - App directly: `http://localhost:${SYSTEM_PORT}` (default `9090`)
  - DB directly: `localhost:${DB_PORT}` (default `5432`)
  - Nginx inside the tester is not exposed on the host — the test only verifies it installs and configures correctly

> **Windows note:** Run from WSL, not PowerShell. On Windows, `${PWD}` expands to `C:\...` which breaks Docker volume parsing (colon collision) and produces paths the host daemon can't find. WSL gives you `/mnt/c/...` — a Linux-style path that Docker Desktop handles correctly on both sides.

## API Endpoints

Rate limit: **100 requests per 60 seconds** (sliding window).

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/health` | — | Health check |
| `GET` | `/metrics` | ✓ | Prometheus metrics |
| `POST` | `/api/v1/providers/search` | — | Search providers with filters |
| `GET` | `/api/v1/providers/filters` | — | Get min/max filter ranges |
| `POST` | `/api/v1/providers` | — | Submit provider telemetry |
| `GET` | `/api/v1/providers` | ✓ | Get latest telemetry for all providers |
| `POST` | `/api/v1/contracts/statuses` | — | Get storage contract statuses |
| `POST` | `/api/v1/benchmarks` | — | Submit benchmark data |

### Authorization

Protected endpoints (`✓`) require an `Authorization` header:

```
Authorization: Bearer <raw-token>
```

The server validates the token by computing its MD5 hash and comparing it against `SYSTEM_ACCESS_TOKENS` in `.env`. To add a token:

```bash
echo -n "your-secret-token" | md5sum
# copy the hash into SYSTEM_ACCESS_TOKENS in .env
```

Multiple tokens are comma-separated: `SYSTEM_ACCESS_TOKENS=hash1,hash2`

## Workers

The application runs several background workers:
- **Providers Master**: Manages provider lifecycle, health checks, and stored file availability
- **Telemetry Worker**: Processes incoming telemetry data
- **Cleaner Worker**: Removes stale data from the database

## License

Apache-2.0

This project was created by order of a TON Foundation community member.
