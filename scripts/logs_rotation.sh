#!/bin/bash

APP_NAME="mytonprovider.app"
LOG_DIR="/var/log/$APP_NAME"
LOG_FILE="$LOG_DIR/$APP_NAME.log"
LOGROTATE_CONF="/etc/logrotate.d/$APP_NAME"

echo "Setting up log rotation for $APP_NAME..."

mkdir -p "$LOG_DIR"
chown -R "$NEWSUDOUSER:$NEWSUDOUSER" "$LOG_DIR"
touch "$LOG_FILE"
chown "$NEWSUDOUSER:$NEWSUDOUSER" "$LOG_FILE"
chmod 700 "$LOG_DIR"
chmod 600 "$LOG_FILE"
bash -c "cat > $LOGROTATE_CONF" <<EOF
$LOG_FILE {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    dateext
    dateformat -%Y-%m-%d
}
EOF

echo "Done!"
