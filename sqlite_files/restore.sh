#!/bin/bash

DB_FILE="./blog.db"
BACKUP_FILE="./backup.sql"

sqlite3 "$DB_FILE" <<EOF
.bail off
.read $BACKUP_FILE
EOF

echo "Restore completed."
