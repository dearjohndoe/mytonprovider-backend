#!/bin/bash

# Use it if server is not configured for SSH public key authentication
# This script sets up a secure SSH connection to a remote server by copying the local RSA public key to the remote server's authorized keys.
# It also configures the remote server to disable password authentication and enable public key authentication.
# Usage: USERNAME=<username> HOST=<host> PASSWORD=<password> ./init_server_connection.sh

if [ -z "$USERNAME" ] || [ -z "$HOST" ] || [ -z "$PASSWORD" ]; then
  echo "❌ Missing required environment variables"
  echo ""
  echo "Usage example: USERNAME=root HOST=123.45.67.89 PASSWORD=yourpassword ./init_server_connection.sh"
  exit 1
fi

if [[ "$PASSWORD" =~ [\'\"\`\$\\\;\|] ]]; then
  echo "❌ Password contains characters: ' \" \` \$ \\ ; |"
  echo "Please use a password without these special characters for correct script execution."
  exit 1
fi

if ! command -v sshpass &> /dev/null; then
  echo "❌ sshpass not found, please install it first."
  echo "You can install it using: sudo apt-get install sshpass"
  exit 1
fi

ESCAPED_PASSWORD=$(printf '%q' "$PASSWORD")

if [ "$USERNAME" = "root" ]; then
  SSH_DIR="/root/.ssh"
else
  SSH_DIR="/home/$USERNAME/.ssh"
fi

if [ ! -f ~/.ssh/id_rsa.pub ]; then
  echo "RSA key not found, generating..."
  mkdir -p ~/.ssh
  ssh-keygen -t rsa -b 2048 -f ~/.ssh/id_rsa -N ""
fi

PUBLIC_KEY=$(cat ~/.ssh/id_rsa.pub)

sshpass -p "$ESCAPED_PASSWORD" ssh -o StrictHostKeyChecking=no -tt "$USERNAME"@"$HOST" << EOF
mkdir -p $SSH_DIR
chmod 700 $SSH_DIR
echo "$PUBLIC_KEY" >> $SSH_DIR/authorized_keys
chmod 600 $SSH_DIR/authorized_keys
echo "SSH keys setup completed"
exit
EOF

SSH_RESULT=$?
if [ $SSH_RESULT -ne 0 ]; then
  echo "❌ Failed to setup SSH keys on remote server (exit code: $SSH_RESULT)"
  echo "This might be due to:"
  echo "  - Wrong username, host, or password"
  echo "  - SSH connection issues"
  echo "  - Permission problems on remote server"
  exit 1
fi

echo "✅ SSH keys copied successfully"

sshpass -p "$ESCAPED_PASSWORD" ssh -o StrictHostKeyChecking=no -tt "$USERNAME"@"$HOST" << EOF

sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#*ChallengeResponseAuthentication.*/ChallengeResponseAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#*UsePAM.*/UsePAM no/' /etc/ssh/sshd_config
sed -i 's/^#*PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config

systemctl restart ssh || systemctl restart sshd || service ssh restart || service sshd restart
echo "SSH config updated and service restarted"
exit
EOF

SSH_CONFIG_RESULT=$?
if [ $SSH_CONFIG_RESULT -ne 0 ]; then
  echo "⚠️  Warning: Failed to update SSH config (exit code: $SSH_CONFIG_RESULT)"
  echo "You may need to manually disable password authentication"
else
  echo "✅ SSH configuration updated successfully"
fi

sleep 5

if ssh -o BatchMode=yes -o ConnectTimeout=15 -o StrictHostKeyChecking=no "$USERNAME"@"$HOST" "echo 'SSH key authentication successful'" 2>/dev/null; then
  echo "SSH key authentication is working as expected."
  exit 0
fi

echo "❌ SSH key authentication failed."
exit 1
