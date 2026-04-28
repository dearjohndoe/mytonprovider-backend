#!/bin/bash

# Bootstraps SSH key auth on a fresh server when only password access is
# available. Copies the local public key to the remote authorized_keys and
# disables password authentication.
#
# Usage: USERNAME=root HOST=1.2.3.4 PASSWORD=yourpassword ./init_server_connection.sh

set -e

if [ -z "$USERNAME" ] || [ -z "$HOST" ] || [ -z "$PASSWORD" ]; then
    echo "❌ Missing USERNAME / HOST / PASSWORD"
    echo "Usage: USERNAME=root HOST=1.2.3.4 PASSWORD=yourpassword $0"
    exit 1
fi

if ! command -v sshpass &>/dev/null; then
    echo "❌ sshpass not found. Install: sudo apt-get install sshpass"
    exit 1
fi

KEY="${HOME}/.ssh/id_ed25519"
KEY_PUB="${KEY}.pub"
if [ ! -f "$KEY_PUB" ]; then
    if [ -f "${HOME}/.ssh/id_rsa.pub" ]; then
        # Prefer existing RSA key over generating a new one
        KEY_PUB="${HOME}/.ssh/id_rsa.pub"
    else
        echo "Generating ed25519 SSH key..."
        mkdir -p "${HOME}/.ssh"
        ssh-keygen -t ed25519 -f "$KEY" -N ""
    fi
fi

PUBKEY=$(cat "$KEY_PUB")

if [ "$USERNAME" = "root" ]; then
    SSH_DIR="/root/.ssh"
else
    SSH_DIR="/home/$USERNAME/.ssh"
fi

# sshpass -e reads from $SSHPASS — avoids shell-quoting the password twice
export SSHPASS="$PASSWORD"

sshpass -e ssh -o StrictHostKeyChecking=no -o PubkeyAuthentication=no \
    "$USERNAME@$HOST" bash <<EOF
set -e
mkdir -p "$SSH_DIR"
chmod 700 "$SSH_DIR"
grep -qxF '$PUBKEY' "$SSH_DIR/authorized_keys" 2>/dev/null \
    || echo '$PUBKEY' >> "$SSH_DIR/authorized_keys"
chmod 600 "$SSH_DIR/authorized_keys"
echo "SSH key installed."
EOF

if [ $? -ne 0 ]; then
    echo "❌ Failed to install public key on remote (check creds / connectivity)."
    unset SSHPASS
    exit 1
fi
echo "✅ SSH keys copied successfully."

sshpass -e ssh -o StrictHostKeyChecking=no -o PubkeyAuthentication=no \
    "$USERNAME@$HOST" bash <<'EOF'
set -e
sed -i -E 's/^#?\s*PasswordAuthentication\s+\S+/PasswordAuthentication no/'                  /etc/ssh/sshd_config
sed -i -E 's/^#?\s*ChallengeResponseAuthentication\s+\S+/ChallengeResponseAuthentication no/' /etc/ssh/sshd_config
sed -i -E 's/^#?\s*UsePAM\s+\S+/UsePAM no/'                                                  /etc/ssh/sshd_config
sed -i -E 's/^#?\s*PubkeyAuthentication\s+\S+/PubkeyAuthentication yes/'                     /etc/ssh/sshd_config

systemctl restart ssh 2>/dev/null || systemctl restart sshd 2>/dev/null \
    || service ssh restart 2>/dev/null || service sshd restart 2>/dev/null || true
echo "SSH config updated."
EOF

unset SSHPASS

sleep 3
if ssh -o BatchMode=yes -o ConnectTimeout=15 -o StrictHostKeyChecking=no \
    "$USERNAME@$HOST" "echo ok" >/dev/null 2>&1; then
    echo "✅ SSH key authentication verified."
    exit 0
fi

echo "❌ SSH key authentication failed — check sshd config on the host."
exit 1
