#!/bin/bash

# Run this script as root!
# This script sets up a secure server environment by installing necessary packages,
# configuring security settings, creating a new sudo user, and disabling root login.
# Usage: NEWFRONTENDUSER=<username> NEWSUDOUSER=<username> PASSWORD=<password> ./secure_server.sh

if [ "$EUID" -ne 0 ]; then
  echo "❌ Please run as root"
  exit 1
fi

svc_restart() {
    if [ -d /run/systemd/system ] && command -v systemctl &>/dev/null; then
        systemctl restart "$1"
    elif command -v service &>/dev/null; then
        service "$1" restart || service "$1" start || true
    else
        echo "No init system detected — skipping 'restart $1'."
    fi
}

if [ -z "$NEWSUDOUSER" ] || [ -z "$PASSWORD" ]; then
  echo "❌ Missing required environment variables"
  echo ""
  echo "Usage: NEWFRONTENDUSER=<username> NEWSUDOUSER=<username> PASSWORD=<password> $0"
  echo "Example: NEWFRONTENDUSER=frontend NEWSUDOUSER=johndoe PASSWORD=yournewsecurepassword $0"
  exit 1
fi

apt-get update
apt-get -y upgrade
apt-get -y install unattended-upgrades fail2ban ufw sudo

# Auto sec updates
echo "Setting up automatic security updates..."
dpkg-reconfigure unattended-upgrades

# Configure UFW
echo "Configuring UFW..."
ufw default deny incoming
ufw default allow outgoing
ufw allow out 53/udp
ufw allow out 53/tcp
ufw allow out 80/tcp
ufw allow out 443/tcp
ufw allow out 123/udp
ufw allow 80/tcp
ufw allow 16167/udp
ufw allow 123/tcp
ufw allow 22/tcp
ufw --force enable || echo "⚠️  'ufw enable' failed (likely missing kernel modules in this environment) — skipping."

# Fail2ban configuration
echo "Configuring Fail2ban..."
cat <<EOL > /etc/fail2ban/jail.local
[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 5
bantime = 3600
findtime = 600
[ufw]
enabled = true
port = 80,16167,123,22
filter = ufw
logpath = /var/log/ufw.log
maxretry = 5
bantime = 3600
findtime = 600
EOL
svc_restart fail2ban

# Backend root user
echo "Creating new sudo user $NEWSUDOUSER..."
adduser --disabled-password --gecos "" "$NEWSUDOUSER"
usermod -aG sudo "$NEWSUDOUSER"
mkdir -p /home/"$NEWSUDOUSER"/.ssh
mkdir -p /opt/provider
chmod 700 /home/"$NEWSUDOUSER"/.ssh
chown "$NEWSUDOUSER":"$NEWSUDOUSER" /home/"$NEWSUDOUSER"/.ssh
if [ -f /root/.ssh/authorized_keys ]; then
  cp /root/.ssh/authorized_keys /home/"$NEWSUDOUSER"/.ssh/
  chmod 600 /home/"$NEWSUDOUSER"/.ssh/authorized_keys
  chown "$NEWSUDOUSER":"$NEWSUDOUSER" /home/"$NEWSUDOUSER"/.ssh/authorized_keys
elif [ "${SKIP_CLONE:-false}" = "true" ]; then
  echo "⚠️  /root/.ssh/authorized_keys not found — skipping key copy for $NEWSUDOUSER (test mode)."
else
  echo "❌ /root/.ssh/authorized_keys not found. Run init_server_connection.sh first, or place your public key there."
  exit 1
fi
chown -R "$NEWSUDOUSER":"$NEWSUDOUSER" /opt/provider
mkdir -p /var/log/mytonprovider.app
chown -R "$NEWSUDOUSER":"$NEWSUDOUSER" /var/log/mytonprovider.app
echo "$NEWSUDOUSER:$PASSWORD" | chpasswd

# Frontend user
if [ -n "$NEWFRONTENDUSER" ]; then
echo "Creating frontend user $NEWFRONTENDUSER..."
adduser --disabled-password --gecos "" "$NEWFRONTENDUSER"
usermod --lock "$NEWFRONTENDUSER"
mkdir -p /home/"$NEWFRONTENDUSER"/.ssh /tmp/frontend
chmod 700 /home/"$NEWFRONTENDUSER"/.ssh
chown "$NEWFRONTENDUSER":"$NEWFRONTENDUSER" /home/"$NEWFRONTENDUSER"/.ssh /tmp/frontend
if [ -f /root/.ssh/authorized_keys ]; then
  cp /root/.ssh/authorized_keys /home/"$NEWFRONTENDUSER"/.ssh/
  chmod 600 /home/"$NEWFRONTENDUSER"/.ssh/authorized_keys
  chown "$NEWFRONTENDUSER":"$NEWFRONTENDUSER" /home/"$NEWFRONTENDUSER"/.ssh/authorized_keys
else
  echo "⚠️  /root/.ssh/authorized_keys not found — skipping key copy for $NEWFRONTENDUSER."
fi

chown -R "$NEWFRONTENDUSER":"$NEWFRONTENDUSER" /var/www
fi

echo "Disabling root login..."
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
  sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
  ALLOWED_USERS="$NEWSUDOUSER"
  ALLOWED_USERS="$ALLOWED_USERS $NEWFRONTENDUSER"
  echo "AllowUsers $ALLOWED_USERS" | sudo tee -a /etc/ssh/sshd_config > /dev/null

  svc_restart ssh || svc_restart sshd || true
else
  echo "⚠️  /etc/ssh/sshd_config not found — skipping SSH hardening."
fi
