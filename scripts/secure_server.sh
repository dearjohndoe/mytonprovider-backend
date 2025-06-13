#!/bin/bash

# Check args
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi
# NEWSUDOUSER=JohnDoe PASSWORD=yourpassword ./secure_server.sh
if [ -z "$NEWSUDOUSER" ] || [ -z "$PASSWORD" ]; then
  echo "Usage: NEWSUDOUSER=<username> PASSWORD=<password> $0"
  echo "Example: NEWSUDOUSER=JohnDoe PASSWORD=yourpassword $0"
  exit 1
fi

apt update
apt -y upgrade
apt -y install unattended-upgrades fail2ban ufw sudo

# Auto sec updates
echo "Setting up automatic security updates..."
dpkg-reconfigure unattended-upgrades

# Configure UFW
echo "Configuring UFW..."
ufw default deny incoming
ufw default deny outgoing
ufw allow 80/tcp
ufw allow 16167/tcp
ufw allow 5432/tcp
ufw allow 123/tcp
ufw allow 22/tcp
ufw enable

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
port = 80,16167,5432,123,22
filter = ufw
logpath = /var/log/ufw.log
maxretry = 5
bantime = 3600
findtime = 600
EOL
systemctl restart fail2ban

# Disable root
echo "Creating new sudo user $NEWSUDOUSER..."
adduser --disabled-password --gecos "" "$NEWSUDOUSER"
usermod -aG sudo "$NEWSUDOUSER"
mkdir -p /home/"$NEWSUDOUSER"/.ssh
chmod 700 /home/"$NEWSUDOUSER"/.ssh
chown "$NEWSUDOUSER":"$NEWSUDOUSER" /home/"$NEWSUDOUSER"/.ssh
cp /root/.ssh/authorized_keys /home/"$NEWSUDOUSER"/.ssh/
chmod 600 /home/"$NEWSUDOUSER"/.ssh/authorized_keys
chown "$NEWSUDOUSER":"$NEWSUDOUSER" /home/"$NEWSUDOUSER"/.ssh/authorized_keys
chown -R "$NEWSUDOUSER":"$NEWSUDOUSER" /opt/provider
chown -R "$NEWSUDOUSER":"$NEWSUDOUSER" /var/www/mytonprovider.org
chown -R "$NEWSUDOUSER":"$NEWSUDOUSER" /var/log/mytonprovider.app
echo "$NEWSUDOUSER:$PASSWORD" | chpasswd

echo "Disabling root login..."
sed -i 's/^PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
echo "AllowUsers $NEWSUDOUSER" | sudo tee -a /etc/ssh/sshd_config > /dev/null
systemctl restart sshd
