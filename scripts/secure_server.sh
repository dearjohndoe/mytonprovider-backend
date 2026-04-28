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

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" upgrade
apt-get -y install unattended-upgrades fail2ban ufw sudo

# Auto sec updates (non-interactive)
echo "Setting up automatic security updates..."
echo 'unattended-upgrades unattended-upgrades/enable_auto_updates boolean true' \
    | debconf-set-selections
dpkg-reconfigure -f noninteractive unattended-upgrades

# Configure UFW
echo "Configuring UFW..."
ufw default deny incoming
ufw default allow outgoing
ufw allow out 53/udp
ufw allow out 53/tcp
ufw allow out 80/tcp
ufw allow out 443/tcp
ufw allow out 123/udp
ufw allow 22/tcp     comment 'SSH'
ufw allow 80/tcp     comment 'HTTP (nginx)'
ufw allow 443/tcp    comment 'HTTPS (nginx)'

# Trust tailscale interface — agents on other VPS reach redis/postgres via it.
if ip link show tailscale0 &>/dev/null; then
    echo "Allowing all traffic on tailscale0..."
    ufw allow in on tailscale0 comment 'tailscale'
fi

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
port = 22,80,443
filter = ufw
logpath = /var/log/ufw.log
maxretry = 5
bantime = 3600
findtime = 600
EOL
svc_restart fail2ban

# Backend root user
echo "Creating new sudo user $NEWSUDOUSER..."
if ! id "$NEWSUDOUSER" &>/dev/null; then
    useradd -m -s /bin/bash "$NEWSUDOUSER"
fi
usermod -aG sudo "$NEWSUDOUSER"
usermod -aG docker "$NEWSUDOUSER" 2>/dev/null || true
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
if ! id "$NEWFRONTENDUSER" &>/dev/null; then
    useradd -m -s /bin/bash "$NEWFRONTENDUSER"
fi
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
  sed -i -E 's/^#?\s*PermitRootLogin\s+\S+/PermitRootLogin no/'                                /etc/ssh/sshd_config
  sed -i -E 's/^#?\s*PasswordAuthentication\s+\S+/PasswordAuthentication no/'                  /etc/ssh/sshd_config
  sed -i -E 's/^#?\s*ChallengeResponseAuthentication\s+\S+/ChallengeResponseAuthentication no/' /etc/ssh/sshd_config
  sed -i -E 's/^#?\s*PubkeyAuthentication\s+\S+/PubkeyAuthentication yes/'                     /etc/ssh/sshd_config

  ALLOWED_USERS="$NEWSUDOUSER"
  [ -n "$NEWFRONTENDUSER" ] && ALLOWED_USERS="$ALLOWED_USERS $NEWFRONTENDUSER"
  # Idempotent — replace any existing AllowUsers line, otherwise append.
  if grep -qE '^AllowUsers\s' /etc/ssh/sshd_config; then
      sed -i -E "s/^AllowUsers\s.*/AllowUsers $ALLOWED_USERS/" /etc/ssh/sshd_config
  else
      echo "AllowUsers $ALLOWED_USERS" >> /etc/ssh/sshd_config
  fi

  svc_restart ssh || svc_restart sshd || true
else
  echo "⚠️  /etc/ssh/sshd_config not found — skipping SSH hardening."
fi
