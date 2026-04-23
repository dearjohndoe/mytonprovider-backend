#!/bin/bash

# Local development setup script.
# Starts all services (db, migrations, app) via Docker Compose.
#
# Requirements: Docker with compose plugin (docker compose) or standalone docker-compose
#
# Usage:
#   bash scripts/setup_local.sh

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"
ENV_EXAMPLE="$ROOT_DIR/.env.example"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

check_docker() {
    if ! command -v docker &>/dev/null; then
        print_error "Docker not found. Please install Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    if ! docker info &>/dev/null; then
        print_error "Docker daemon is not running. Please start Docker."
        exit 1
    fi
}

detect_compose() {
    if docker compose version &>/dev/null 2>&1; then
        echo "docker compose"
    elif command -v docker-compose &>/dev/null; then
        echo "docker-compose"
    else
        print_error "Docker Compose not found. Please install Docker Desktop or the compose plugin."
        exit 1
    fi
}

ensure_env_file() {
    if [ ! -f "$ENV_FILE" ]; then
        if [ ! -f "$ENV_EXAMPLE" ]; then
            print_error ".env.example not found. Cannot create .env."
            exit 1
        fi
        print_status "Creating .env from .env.example..."
        # DB_HOST must point to the Docker service name, not localhost
        sed 's/^DB_HOST=localhost/DB_HOST=db/' "$ENV_EXAMPLE" > "$ENV_FILE"
        print_success "Created .env — review and adjust values if needed."
    else
        print_status ".env already exists, skipping."
    fi
}

check_config_path() {
    local config_path
    config_path=$(grep -E '^CONFIG_PATH=' "$ENV_FILE" | cut -d= -f2-)

    if [ -z "$config_path" ]; then
        print_status "CONFIG_PATH not set in .env, adding default..."
        echo "CONFIG_PATH=config/dev.yaml" >> "$ENV_FILE"
        config_path="config/dev.yaml"
    fi

    # Resolve path: relative is relative to the project root
    local resolved="$ROOT_DIR/$config_path"
    if [[ "$config_path" = /* ]]; then
        resolved="$config_path"
    fi

    if [ ! -f "$resolved" ]; then
        print_error "Config file not found: $resolved"
        print_error "Set CONFIG_PATH in .env to a valid .yaml file under config/."
        exit 1
    fi

    print_status "Using config: $resolved"
}

main() {
    print_status "Setting up local development environment..."

    check_docker

    COMPOSE=$(detect_compose)
    print_status "Using: $COMPOSE"

    ensure_env_file
    check_config_path

    print_status "Building and starting all services..."
    $COMPOSE -f "$ROOT_DIR/docker-compose.yml" up -d --build

    print_success "All services started."
    echo ""
    echo "View logs:"
    echo "  $COMPOSE -f docker-compose.yml logs -f app"
    echo "  $COMPOSE -f docker-compose.yml logs -f db"
    echo ""
    echo "Stop services:"
    echo "  $COMPOSE -f docker-compose.yml down"
    echo ""
    echo "Rebuild after code changes:"
    echo "  $COMPOSE -f docker-compose.yml up -d --build app"
}

main "$@"
