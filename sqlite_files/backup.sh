#!/bin/bash

DB_FILE="./blog.db"
BACKUP_FILE="./backup.sql"

sqlite3 "$DB_FILE" ".dump --data-only" > "$BACKUP_FILE"

echo "Data-only backup saved to $BACKUP_FILE"
