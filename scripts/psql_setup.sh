#!/bin/bash

PG_CONF="/etc/postgresql/${PG_VERSION}/main/postgresql.conf"
PG_HBA="/etc/postgresql/${PG_VERSION}/main/pg_hba.conf"

echo "Adding PostgreSQL APT repository..."
apt-get update
curl -s https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor > /etc/apt/trusted.gpg.d/postgresql.gpg
sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'

echo "Installing PostgreSQL $PG_VERSION..."
apt-get update
apt-get upgrade -y
apt-get install -y postgresql-$PG_VERSION

# check is postgresql is installed
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL $PG_VERSION installation failed"
    exit 1
fi

# configure psql
sed -i 's/^host\s\+all\s\+all\s\+127\.0\.0\.1\/32\s\+.*/host all all 127.0.0.1\/32 md5/' "$PG_HBA"
sed -i "s/^#listen_addresses =.*/listen_addresses = '*'/" "$PG_CONF"
sed -i "s/^listen_addresses = 'localhost'/listen_addresses = '*'/" "$PG_CONF"
grep -q "0.0.0.0/0" "$PG_HBA" || echo "host    all             all             0.0.0.0/0               md5" >> "$PG_HBA"
systemctl restart postgresql

# create user
echo "Creating PostgreSQL user and database..."
su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname = '$PG_USER';\"" | grep -q 1 || \
  su - postgres -c "psql -c \"CREATE USER $PG_USER WITH PASSWORD '$PG_PASSWORD';\""

su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname = '$PG_DB';\"" | grep -q 1 || \
  su - postgres -c "psql -c \"CREATE DATABASE $PG_DB OWNER $PG_USER;\""

echo "Checking connection..."
if ! PGPASSWORD="$PG_PASSWORD" psql -h 127.0.0.1 -U "$PG_USER" -d "$PG_DB" -c '\conninfo'; then
    echo "❌ Failed to connect to PostgreSQL database"
    exit 1
fi

echo "✅ PostgreSQL $PG_VERSION ready to use."
