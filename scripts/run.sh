#!/bin/bash

cd /opt/provider/my

env $(cat config.env | xargs) ./mtpo-backend >> /var/log/mytonprovider.app/mytonprovider.app.log 2>&1 &

sleep 2

if pgrep -f "./mtpo-backend" > /dev/null; then
    echo "✅ Backend application started successfully."
else
    echo "❌ Failed to start backend application."
    exit 1
fi
