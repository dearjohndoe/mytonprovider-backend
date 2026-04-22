#!/bin/bash

cd /opt/provider

mkdir -p /var/log/mytonprovider.app

env $(grep -v '^\s*$' config.env | grep -v '=$' | xargs) ./mtpo-backend >> /var/log/mytonprovider.app/mytonprovider.app.log 2>&1 &

sleep 5

if pgrep -f "./mtpo-backend" > /dev/null; then
    echo "✅ Backend application started successfully."
else
    echo "❌ Failed to start backend application."
    exit 1
fi