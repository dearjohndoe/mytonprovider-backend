#!/bin/bash

set -e

REPO_DIR="mytonprovider-org"
REPO_URL="https://github.com/dearjohndoe/mytonprovider-org.git"

mkdir -p /tmp/frontend
cd /tmp/frontend

if [ -d "$REPO_DIR" ]; then
    echo "Repository directory '$REPO_DIR' found. Pulling latest changes."
    cd "$REPO_DIR"
    git reset --hard
    git pull
else
    echo "Cloning repository from $REPO_URL."
    git clone "$REPO_URL"
    cd "$REPO_DIR"
fi

# Hard replace backend host in lib/api.ts
if [ "$INSTALL_SSL" = "true" ]; then
    PROTOCOL="https"
else
    PROTOCOL="http"
fi

echo "Replacing backend host in lib/api.ts with $PROTOCOL://$HOST"
sed -i "s|https://mytonprovider.org|$PROTOCOL://$HOST|g" lib/api.ts


echo "Installing npm dependencies..."
npm install --legacy-peer-deps

echo "Building the project..."
npm run build

DOMAIN="${DOMAIN:-mytonprovider.org}"
# Use IP address if no domain is provided or if DOMAIN is an IP
if [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    SITE_NAME="ip-${DOMAIN//./-}"
else
    SITE_NAME="$DOMAIN"
fi

WEB_DIR="/var/www/$SITE_NAME"
BUILD_DIR="out"

rm -rf "$WEB_DIR"
mkdir -p "$WEB_DIR"
cp -r "$BUILD_DIR"/* "$WEB_DIR/"

echo "Frontend deployment completed successfully."
