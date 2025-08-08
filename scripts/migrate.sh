#!/bin/bash

# Read the .secrets/pgadmin file
username=${POSTGRES_USER:-"pgadmin"}
secret=`cat ${POSTGRES_PASSWORD_FILE:-.secrets/postgres_admin}`
host=${POSTGRES_HOST:-"localhost"}
port=${POSTGRES_PORT:-"5432"}
database=${POSTGRES_APP_DB:-"app"}
sslmode="${POSTGRES_SSLMODE:-"disable"}"
migrations=${MIGRATIONS_PATH:-"./migrations"}
conn_str="postgres://$username:$secret@$host:$port/$database?sslmode=$sslmode"

if [ "$1" = "down" ]; then
    bash -c "migrate -path $migrations -database $conn_str down"
elif [ "$1" = "up" ]; then
    bash -c "migrate -path $migrations -database $conn_str up"
elif [ "$1" = "force" ]; then
    bash -c "migrate -path $migrations -database $conn_str force $2"
else
    echo "Usage: $0 [up|down|force <version>]"
    exit 1
fi