#!/bin/bash

# Main server setup script that automates the entire server configuration process
# This script runs directly on the target server, downloads all necessary scripts
# from GitHub, installs PostgreSQL, configures Nginx, sets up log rotation,
# installs the backend application, secures the server, and initializes the database.
#
# Usage: Download and run with environment variables:
# wget https://raw.githubusercontent.com/dearjohndoe/mytonprovider-backend/master/scripts/setup_server.sh
# chmod +x setup_server.sh
# PG_USER=<pguser> PG_PASSWORD=<pgpassword> PG_DB=<database> \
# NEWFRONTENDUSER=<frontenduser> \
# NEWSUDOUSER=<newuser> NEWUSER_PASSWORD=<newpassword> \
# DOMAIN=<domain> INSTALL_SSL=<true|false> APP_USER=<appuser> \
# ./setup_server.sh

set -e

PG_VERSION="15"
GITHUB_REPO="dearjohndoe/mytonprovider-backend"
GITHUB_BRANCH="master"
SCRIPTS_BASE_URL="https://raw.githubusercontent.com/$GITHUB_REPO/$GITHUB_BRANCH/scripts"
DB_BASE_URL="https://raw.githubusercontent.com/$GITHUB_REPO/$GITHUB_BRANCH/db"
WORK_DIR="/tmp/provider"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_required_vars() {
    local required_vars=(
        "PG_USER"
        "PG_PASSWORD"
        "PG_DB"
        "NEWSUDOUSER"
        "NEWFRONTENDUSER"
        "NEWUSER_PASSWORD"
    )
    
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        print_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        echo ""
        echo "Usage example:"
        echo "PG_USER=pguser PG_PASSWORD=secret PG_DB=providerdb \\"
        echo "NEWFRONTENDUSER=frontend \\"
        echo "NEWSUDOUSER=johndoe NEWUSER_PASSWORD=newsecurepassword \\"
        echo "DOMAIN=mytonprovider.org INSTALL_SSL=true \\"
        echo "./setup_server.sh"
        echo ""
        echo "Note: DOMAIN is optional. If not provided, will use server's hostname/IP."
        echo "      SSL certificates require a domain name."
        exit 1
    fi
}

setup_work_directory() {
    print_status "Setting up work directory..."

    if [ -d "mytonprovider-backend" ]; then
        echo "Repository exists, pulling latest changes..."
        cd mytonprovider-backend || exit 1
        git pull origin master
    else
        echo "Cloning repository..."
        git clone https://github.com/dearjohndoe/mytonprovider-backend
    fi
    
    print_success "Work directory set up successfully."
}

execute_script() {
    local script_name=$1
    
    if [[ ! -f "$script_name" ]]; then
        print_error "Script not found: $script_name"
        exit 1
    fi
    
    local env_vars=""
    local vars_to_pass=(
        "PG_VERSION" "PG_USER" "PG_PASSWORD" "PG_DB"
        "NEWFRONTENDUSER" "WORK_DIR"
        "NEWSUDOUSER" "NEWUSER_PASSWORD" "DOMAIN" "INSTALL_SSL"
    )
    
    for var in "${vars_to_pass[@]}"; do
        if [[ -n "${!var}" ]]; then
            export $var="${!var}"
        fi
    done

    if ! bash "$script_name"; then
        print_error "Script $script_name failed with exit code $?"
        exit 1
    fi
}

install_deps() {
    print_status "Installing required dependencies..."
    
    apt-get update
    apt-get upgrade -y
    apt-get install -y wget curl gnupg lsb-release git

    if ! command -v go &> /dev/null && [ ! -f /usr/local/go/bin/go ]; then
        print_status "Installing Go..."
        wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
        tar -C /usr/local -xzf go1.24.5.linux-amd64.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
        rm go1.24.5.linux-amd64.tar.gz
    fi

    export PATH=$PATH:/usr/local/go/bin

    if ! command -v node &> /dev/null; then
        wget -qO- https://deb.nodesource.com/setup_20.x | bash -
        apt-get install -y nodejs
    fi
}

get_server_info() {
    HOST=$(hostname -I | awk '{print $1}')
    if [[ -z "$HOST" ]]; then
        HOST=$(hostname -f)
    fi
    
    print_status "Detected server information:"
    echo "Server IP/Hostname: $HOST"
}

main() {
    print_status "Starting server setup process..."
    
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root"
        echo "Please run: sudo $0"
        exit 1
    fi

    mkdir -p "$WORK_DIR"
    cd "$WORK_DIR" || exit 1

    check_required_vars

    install_deps
    
    get_server_info
    
    DOMAIN="${DOMAIN:-$HOST}"
    
    print_status "All required environment variables are set"
    echo "Server IP/Hostname: $HOST"
    echo "New sudo user: $NEWSUDOUSER"
    echo "New frontend user: $NEWFRONTENDUSER"
    echo "PostgreSQL version: $PG_VERSION"
    echo "PostgreSQL database: $PG_DB"
    echo "Domain/IP: $DOMAIN"
    echo ""
    
    print_status "Step 1: Downloading scripts and configuration files..."
    setup_work_directory
    cd "$WORK_DIR/mytonprovider-backend/scripts" || exit 1
    
    print_status "Step 2: Setting up PostgreSQL..."
    execute_script "psql_setup.sh"
    
    print_status "Step 3: Disabling postgres user remote access..."
    execute_script "ib_disable_postgres_user.sh"
    
    print_status "Step 4: Initializing database..."
    execute_script "init_db.sh"
    
    print_status "Step 5: Setting up Nginx..."
    execute_script "setup_nginx.sh"
    
    print_status "Step 6: Setting up log rotation..."
    execute_script "logs_rotation.sh"
    
    print_status "Step 7: Securing the server..."
    export PASSWORD="$NEWUSER_PASSWORD"  # secure_server.sh expects PASSWORD env var
    execute_script "secure_server.sh"
    
    print_status "Step 8: Building backend application..."
    execute_script "build_backend.sh"
    
    print_status "Step 9: Running the backend application..."
    su - "$NEWSUDOUSER" -c "cd $WORK_DIR/mytonprovider-backend/scripts && bash run.sh"

    print_status "Step 10: Building and deploying frontend..."
    su - "$NEWFRONTENDUSER" -c "cd $WORK_DIR/mytonprovider-backend/scripts && HOST='$HOST' DOMAIN='$DOMAIN' INSTALL_SSL='$INSTALL_SSL' bash build_frontend.sh"

    print_success "Server setup completed successfully!"
    echo ""
    echo "Summary:"
    echo "✅ All scripts downloaded from GitHub"
    echo "✅ SSH key authentication configured"
    echo "✅ PostgreSQL $PG_VERSION installed and configured"
    echo "✅ Database '$PG_DB' initialized"
    echo "✅ Nginx installed and configured"
    echo "✅ Log rotation configured"
    echo "✅ Backend application installed"
    echo "✅ Server secured with user '$NEWSUDOUSER'"
    echo "✅ Frontend application built and deployed"
    echo "✅ Frontend user created: $NEWFRONTENDUSER"
    echo ""
    echo "You can now connect to your server using:"
    echo "ssh $NEWSUDOUSER@$HOST"
    echo "from there you can also connect as the frontend user using:"
    echo "sudo su $NEWFRONTENDUSER"
    echo ""
    echo "Web services:"
    echo "Website: http://$DOMAIN"
    echo "API: http://$DOMAIN/api/"
    echo "Health check: http://$DOMAIN/health"
    echo "Metrics: http://$DOMAIN/metrics"
    echo ""
    echo "Backend application:"
    echo "Install directory: /opt/provider/my"
    echo "Start service: cd /opt/provider/my && env \$(cat config.env | xargs) ./mtpo-backend >> /var/log/mytonprovider.app/mytonprovider.app.log 2>&1 &"
    echo "View logs: tail -f /var/log/mytonprovider.app/mytonprovider.app.log"
    echo ""
    echo "Database connection details:"
    echo "Host: $HOST"
    echo "Port: 5432"
    echo "Database: $PG_DB"
    echo "User: $PG_USER"
    echo ""
    echo "Cleanup: rm -rf $WORK_DIR"
}

main "$@"
