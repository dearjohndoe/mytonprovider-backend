#!/bin/bash

# Main server setup script that automates the entire server configuration process
# This script connects to a remote server, installs PostgreSQL, configures Nginx,
# sets up log rotation, installs the backend application, secures the server,
# and initializes the database.
#
# Usage: Set all required environment variables and run:
# REMOTEUSER=<username> HOST=<host> PASSWORD=<password> \
# PG_VERSION=<version> PG_USER=<pguser> PG_PASSWORD=<pgpassword> PG_DB=<database> \
# NEWSUDOUSER=<newuser> NEWUSER_PASSWORD=<newpassword> \
# DOMAIN=<domain> INSTALL_SSL=<true|false> APP_USER=<appuser> \
# ./setup_server.sh

set -e

# Color codes for output
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
        "REMOTEUSER"
        "HOST"
        "PASSWORD"
        "PG_VERSION"
        "PG_USER"
        "PG_PASSWORD"
        "PG_DB"
        "NEWSUDOUSER"
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
        echo "REMOTEUSER=root HOST=123.45.67.89 PASSWORD=yourpassword \\"
        echo "PG_VERSION=15 PG_USER=pguser PG_PASSWORD=secret PG_DB=providerdb \\"
        echo "NEWSUDOUSER=johndoe NEWUSER_PASSWORD=newsecurepassword \\"
        echo "DOMAIN=mytonprovider.org INSTALL_SSL=true APP_USER=provideruser \\"
        echo "$0"
        echo ""
        echo "Note: DOMAIN is optional. If not provided, will use HOST value."
        echo "      APP_USER is optional. If not provided, will use 'provideruser'."
        echo "      SSL certificates require a domain name (not IP address)."
        exit 1
    fi
}

execute_script() {
    local script_name=$1
    local script_path="$(dirname "$0")/$script_name"
    
    if [[ ! -f "$script_path" ]]; then
        print_error "Script not found: $script_path"
        exit 1
    fi
    
    print_status "Executing $script_name..."
    
    if bash "$script_path"; then
        print_success "$script_name completed successfully"
    else
        print_error "$script_name failed"
        exit 1
    fi
}

execute_remote_script() {
    local script_name=$1
    local user=${2:-$REMOTEUSER}
    local script_path="$(dirname "$0")/$script_name"
    
    if [[ ! -f "$script_path" ]]; then
        print_error "Script not found: $script_path"
        exit 1
    fi
    
    print_status "Uploading and executing $script_name on remote server as user $user..."
    
    if [[ "$user" == "$REMOTEUSER" ]]; then
        ssh "$user"@"$HOST" "mkdir -p /tmp/server_setup/scripts && chmod -R 777 /tmp/server_setup"
    fi
    
    scp "$script_path" "$user"@"$HOST":/tmp/server_setup/scripts/
    
    if [[ "$script_name" == "init_db.sh" ]]; then
        local db_init_path="$(dirname "$0")/../db/init.sql"
        ssh "$user"@"$HOST" "mkdir -p /tmp/server_setup/db"
        scp "$db_init_path" "$user"@"$HOST":/tmp/server_setup/db/
    fi

    if [[ "$script_name" == "install_backend.sh" ]]; then
        local exe_path="$(dirname "$0")/mtpo-backend"
        local config_path="$(dirname "$0")/config.env"
        scp "$exe_path" "$user"@"$HOST":/opt/provider/my/
        scp "$config_path" "$user"@"$HOST":/opt/provider/my/
    fi
    
    # Pass env
    local env_vars=""
    local vars_to_pass=(
        "PG_VERSION" "PG_USER" "PG_PASSWORD" "PG_DB"
        "NEWSUDOUSER" "NEWUSER_PASSWORD" "DOMAIN" "INSTALL_SSL" "APP_USER"
        "REMOTEUSER" "HOST" "PASSWORD"
    )
    
    for var in "${vars_to_pass[@]}"; do
        if [[ -n "${!var}" ]]; then
            env_vars+=" $var='${!var}'"
        fi
    done
    
    if ssh "$user"@"$HOST" "cd /tmp/server_setup/scripts && env $env_vars bash $script_name"; then
        print_success "$script_name completed successfully on remote server"
    else
        print_error "$script_name failed on remote server"
        exit 1
    fi
}

main() {
    print_status "Starting server setup process..."
    
    # Check if all required environment variables are set
    check_required_vars
    
    # Set default values for optional variables
    DOMAIN="${DOMAIN:-$HOST}"
    APP_USER="${APP_USER:-provideruser}"
    
    print_status "All required environment variables are set"
    echo "Target server: $HOST"
    echo "Remote user: $REMOTEUSER"
    echo "New sudo user: $NEWSUDOUSER"
    echo "PostgreSQL version: $PG_VERSION"
    echo "PostgreSQL database: $PG_DB"
    echo "Domain/IP: $DOMAIN"
    echo "App user: $APP_USER"
    echo ""
    
    print_status "Step 1: Setting up SSH key authentication..."
    execute_script "init_server_connection.sh"
    
    print_status "Step 2: Setting up PostgreSQL..."
    execute_remote_script "psql_setup.sh"
    
    print_status "Step 3: Disabling postgres user remote access..."
    execute_remote_script "ib_disable_postgres_user.sh"
    
    print_status "Step 4: Initializing database..."
    execute_remote_script "init_db.sh"
    
    print_status "Step 5: Setting up Nginx..."
    execute_remote_script "setup_nginx.sh"
    
    print_status "Step 6: Setting up log rotation..."
    execute_remote_script "logs_rotation.sh"
    
    print_status "Step 7: Securing the server..."
    export PASSWORD="$NEWUSER_PASSWORD"  # secure_server.sh expects PASSWORD env var
    execute_remote_script "secure_server.sh"
    
    print_status "Step 8: Installing backend application..."
    execute_script "build_backend.sh"
    execute_remote_script "install_backend.sh" "$NEWSUDOUSER"

    print_status "Step 9: Running the backend application..."
    execute_remote_script "run.sh" "$NEWSUDOUSER"

    print_status "Step 10: Building and deploying frontend..."
    execute_remote_script "build_frontend.sh" "$NEWSUDOUSER"
    
    print_success "Server setup completed successfully!"
    echo ""
    echo "Summary:"
    echo "✅ SSH key authentication configured"
    echo "✅ PostgreSQL $PG_VERSION installed and configured"
    echo "✅ Database '$PG_DB' initialized"
    echo "✅ Nginx installed and configured"
    echo "✅ Log rotation configured"
    echo "✅ Backend application installed"
    echo "✅ Server secured with user '$NEWSUDOUSER'"
    echo ""
    echo "You can now connect to your server using:"
    echo "ssh $NEWSUDOUSER@$HOST"
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
}

main "$@"
