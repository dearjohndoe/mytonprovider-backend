#!/bin/bash

# Main server setup script — configures a fresh server using Docker Compose.
#
# Usage:
# wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/master/scripts/setup_server.sh
# chmod +x setup_server.sh
# DB_HOST=<host> DB_USER=<user> DB_PASSWORD=<password> DB_NAME=<db> \
# MASTER_ADDRESS=<ton-master-wallet> \
# NEWSUDOUSER=<newuser> NEWUSER_PASSWORD=<password> \
# NEWFRONTENDUSER=<frontenduser> \
# DOMAIN=<domain> INSTALL_SSL=<true|false> \
# [TAILSCALE_AUTHKEY=tskey-auth-...] \
# ./setup_server.sh
#
# Optional vars:
#   TAILSCALE_AUTHKEY  — if set, installs tailscale and joins the tailnet
#                        (so agents from other VPS can reach redis/postgres
#                        over the private network).

set -e

GITHUB_REPO="dearjohndoe/mytonprovider-backend"
GITHUB_BRANCH="master"
WORK_DIR="${WORK_DIR:-/opt/provider}"

DB_PORT="${DB_PORT:-5432}"
SYSTEM_PORT="${SYSTEM_PORT:-9090}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status()  { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

check_required_vars() {
    local required_vars=(
        "DB_HOST" "DB_USER" "DB_PASSWORD" "DB_NAME"
        "MASTER_ADDRESS"
        "NEWSUDOUSER" "NEWUSER_PASSWORD"
        "NEWFRONTENDUSER"
    )
    local missing_vars=()
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            missing_vars+=("$var")
        fi
    done
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        print_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do echo "  - $var"; done
        echo ""
        echo "Usage example:"
        echo "DB_USER=pguser DB_PASSWORD=secret DB_NAME=providerdb \\"
        echo "NEWSUDOUSER=johndoe NEWUSER_PASSWORD=newsecurepassword \\"
        echo "NEWFRONTENDUSER=frontend \\"
        echo "DOMAIN=mytonprovider.org INSTALL_SSL=true \\"
        echo "./setup_server.sh"
        exit 1
    fi
}

install_deps() {
    print_status "Installing system dependencies..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" upgrade
    apt-get install -y curl git ca-certificates gnupg lsb-release
}

install_tailscale() {
    [[ -z "$TAILSCALE_AUTHKEY" ]] && return
    if command -v tailscale &>/dev/null; then
        print_status "Tailscale already installed: $(tailscale version | head -n1)"
    else
        print_status "Installing Tailscale..."
        curl -fsSL https://tailscale.com/install.sh | sh
    fi
    if tailscale ip -4 &>/dev/null; then
        print_status "Tailscale already up: $(tailscale ip -4 | head -n1)"
    else
        local hostname_arg=""
        [[ -n "$TAILSCALE_HOSTNAME" ]] && hostname_arg="--hostname=$TAILSCALE_HOSTNAME"
        tailscale up --authkey="$TAILSCALE_AUTHKEY" --ssh=false $hostname_arg
        print_success "Tailscale up: $(tailscale ip -4 | head -n1)"
    fi
}

install_docker() {
    local os_id
    os_id=$(. /etc/os-release && echo "$ID")
    if [[ "$os_id" != "debian" && "$os_id" != "ubuntu" ]]; then
        os_id="ubuntu"
    fi

    local need_install=false
    if ! command -v docker &>/dev/null; then
        need_install=true
    elif ! docker compose version &>/dev/null; then
        print_status "Docker present but 'docker compose' plugin missing; installing compose plugin."
        need_install=true
    fi

    if ! $need_install; then
        print_status "Docker already installed: $(docker --version)"
        return
    fi

    print_status "Installing Docker (repo for $os_id)..."
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL "https://download.docker.com/linux/$os_id/gpg" \
        | gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/$os_id $(lsb_release -cs) stable" \
        > /etc/apt/sources.list.d/docker.list
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
    print_success "Docker installed."
}

clone_repo() {
    print_status "Setting up repository in $WORK_DIR..."
    if [ -d "$WORK_DIR/.git" ]; then
        print_status "Repository exists, fetching $GITHUB_BRANCH..."
        git -C "$WORK_DIR" fetch origin "$GITHUB_BRANCH"
        git -C "$WORK_DIR" checkout "$GITHUB_BRANCH"
        git -C "$WORK_DIR" reset --hard "origin/$GITHUB_BRANCH"
    else
        git clone --branch "$GITHUB_BRANCH" "https://github.com/$GITHUB_REPO" "$WORK_DIR"
    fi
    print_success "Repository ready."
}

create_env_file() {
    print_status "Creating .env file..."
    cat > "$WORK_DIR/.env" <<EOL
DB_HOST=${DB_HOST}
DB_USER=${DB_USER}
MASTER_ADDRESS=${MASTER_ADDRESS}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
DB_PORT=${DB_PORT}
SYSTEM_PORT=${SYSTEM_PORT}
CONFIG_PATH=${CONFIG_PATH:-config/dev.yaml}
EOL
    chmod 600 "$WORK_DIR/.env"
    print_success ".env file created."
}

start_app() {
    print_status "Starting application with Docker Compose..."
    docker compose -f "$WORK_DIR/docker-compose.yml" up -d --build
    print_success "Application started."
}

get_server_info() {
    HOST=$(hostname -I | awk '{print $1}')
    if [[ -z "$HOST" ]]; then
        HOST=$(hostname -f)
    fi
}

execute_script() {
    local script="$WORK_DIR/scripts/$1"
    if [[ ! -f "$script" ]]; then
        print_error "Script not found: $script"
        exit 1
    fi
    local vars_to_pass=(
        "NEWSUDOUSER" "NEWUSER_PASSWORD" "NEWFRONTENDUSER"
        "DOMAIN" "INSTALL_SSL" "HOST" "SYSTEM_PORT"
    )
    for var in "${vars_to_pass[@]}"; do
        [[ -n "${!var}" ]] && export "$var=${!var}"
    done
    if ! bash "$script"; then
        print_error "Script $1 failed"
        exit 1
    fi
}

main() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root"
        exit 1
    fi

    print_status "Starting server setup..."
    check_required_vars
    install_deps
    install_docker
    install_tailscale
    get_server_info
    DOMAIN="${DOMAIN:-$HOST}"

    if [[ "${SKIP_CLONE:-false}" == "true" ]]; then
        print_warning "Step 1: SKIP_CLONE=true — using existing $WORK_DIR."
        if [[ ! -d "$WORK_DIR" ]]; then
            print_error "SKIP_CLONE=true but $WORK_DIR does not exist."
            exit 1
        fi
    else
        print_status "Step 1: Cloning repository..."
        clone_repo
    fi

    print_status "Step 2: Creating application configuration..."
    create_env_file

    if [[ "${SKIP_APP_START:-false}" == "true" ]]; then
        print_warning "Step 3: SKIP_APP_START=true — skipping docker compose up."
    else
        print_status "Step 3: Starting application stack..."
        start_app
    fi

    print_status "Step 4: Setting up Nginx..."
    execute_script "setup_nginx.sh"

    print_status "Step 5: Securing the server..."
    export PASSWORD="$NEWUSER_PASSWORD"
    execute_script "secure_server.sh"

    if [[ "${SKIP_FRONTEND:-false}" == "true" ]]; then
        print_warning "Step 6: SKIP_FRONTEND=true — skipping frontend build."
    else
        print_status "Step 6: Building and deploying frontend..."
        su - "$NEWFRONTENDUSER" -c "cd $WORK_DIR/scripts && HOST='$HOST' DOMAIN='$DOMAIN' INSTALL_SSL='$INSTALL_SSL' bash build_frontend.sh"
    fi

    print_success "Server setup completed successfully!"
    echo ""
    echo "Summary:"
    echo "  Docker Compose stack: running"
    echo "  Nginx: configured"
    echo "  SSH user: $NEWSUDOUSER"
    echo "  Frontend user: $NEWFRONTENDUSER"
    echo "  Domain: $DOMAIN"
    echo ""
    echo "Useful commands:"
    echo "  View logs:    docker compose -f $WORK_DIR/docker-compose.yml logs -f app"
    echo "  Restart app:  docker compose -f $WORK_DIR/docker-compose.yml restart app"
    echo "  Stop all:     docker compose -f $WORK_DIR/docker-compose.yml down"
    echo ""
    echo "Web services:"
    echo "  Website:      http://$DOMAIN"
    echo "  API:          http://$DOMAIN/api/"
    echo "  Health check: http://$DOMAIN/health"
    echo "  Metrics:      http://$DOMAIN/metrics"
}

main "$@"
