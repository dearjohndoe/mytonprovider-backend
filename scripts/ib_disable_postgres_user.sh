#!/bin/bash

PGUSER_TO_BLOCK="postgres"

PG_HBA_PATH="/etc/postgresql/15/main/pg_hba.conf"

cp "$PG_HBA_PATH" "${PG_HBA_PATH}.bak.$(date +%s)"

if grep -qE "^host\s+all\s+${PGUSER_TO_BLOCK}\s+0\.0\.0\.0/0\s+reject" "$PG_HBA_PATH"; then
    echo "Rule already exists."
else
    awk -v user="$PGUSER_TO_BLOCK" '
        BEGIN { inserted = 0 }
        /^host/ && inserted == 0 {
            print "host    all    " user "    0.0.0.0/0    reject"
            inserted = 1
        }
        { print }
    ' "$PG_HBA_PATH" > "${PG_HBA_PATH}.tmp" && mv "${PG_HBA_PATH}.tmp" "$PG_HBA_PATH"

    echo "Rule added."
fi

systemctl restart postgresql

echo "Done!"
