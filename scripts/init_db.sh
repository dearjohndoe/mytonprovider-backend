#!/bin/bash

# This script initializes the database with tables, schemas, functions and triggers from db/init.sql

set -e

SQL_FILE="../db/init.sql"

if [[ -z "$PG_USER" || -z "$PG_PASSWORD" || -z "$PG_DB" ]]; then
    echo "❌ Missing required environment variables"
    echo ""
    echo "Usage:"
    echo "PG_HOST=<host> PG_PORT=<port> PG_USER=<username> PG_PASSWORD=<password> PG_DB=<database> bash init_db.sh"
    echo "Example:"
    echo "PG_HOST=127.0.0.1 PG_PORT=5432 PG_USER=pguser PG_PASSWORD=secret PG_DB=providerdb bash init_db.sh"
    echo ""
    echo "PG_HOST and PG_PORT are optional"
    exit 1
fi

PG_HOST="${PG_HOST:-127.0.0.1}"
PG_PORT="${PG_PORT:-5432}"

echo "Initializing database from $SQL_FILE..."
if PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB" -f "$SQL_FILE"; then
    echo "✅ Database initialization completed successfully"
else
    echo "❌ Database initialization failed"
    exit 1
fi

echo "Done!"
