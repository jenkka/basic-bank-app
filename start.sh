#!/bin/sh

set -e

echo "Running database migration"
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

echo "Starting the application"
exec "$@"
