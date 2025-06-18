#!/bin/bash

# Use it if server is not configured for SSH public key authentication
# This script sets up a secure SSH connection to a remote server by copying the local RSA public key to the remote server's authorized keys.
# It also configures the remote server to disable password authentication and enable public key authentication.
# Usage: REMOTEUSER=<username> HOST=<host> PASSWORD=<password> ./init_server_connection.sh

if [ -z "$REMOTEUSER" ] || [ -z "$HOST" ] || [ -z "$PASSWORD" ]; then
  echo "Usage: REMOTEUSER=<username> HOST=<host> PASSWORD=<password> $0"
  echo "Example: REMOTEUSER=root HOST=123.45.67.89 PASSWORD=yourpassword $0"
  exit 1
fi

SSH_DIR="~/.ssh"
if [ "$REMOTEUSER" = "root" ]; then
  SSH_DIR="/root/.ssh"
fi

if ! command -v sshpass &> /dev/null; then
  echo "sshpass not found, installing..."
  sudo apt-get update && sudo apt-get install -y sshpass
fi

if [ ! -f ~/.ssh/id_rsa.pub ]; then
  echo "RSA key not found, generating..."
  mkdir -p ~/.ssh
  ssh-keygen -t rsa -b 2048 -f ~/.ssh/id_rsa -N ""
fi

sshpass -p "$PASSWORD" ssh -tt "$REMOTEUSER"@"$HOST" << EOF
mkdir -p $SSH_DIR || echo "Failed to create directory $SSH_DIR"
chmod 700 $SSH_DIR || echo "Failed to set permissions for $SSH_DIR"
echo "$(cat ~/.ssh/id_rsa.pub)" >> $SSH_DIR/authorized_keys || echo "Failed to append public key to authorized_keys"
chmod 600 $SSH_DIR/authorized_keys || echo "Failed to set permissions for authorized_keys"
exit
EOF

sshpass -p "$PASSWORD" ssh -tt "$REMOTEUSER"@"$HOST" << EOF
sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#ChallengeResponseAuthentication yes/ChallengeResponseAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#UsePAM yes/UsePAM no/' /etc/ssh/sshd_config
sed -i 's/^#PubkeyAuthentication no/PubkeyAuthentication yes/' /etc/ssh/sshd_config

systemctl restart ssh || systemctl restart sshd || service ssh restart || service sshd restart
exit
EOF