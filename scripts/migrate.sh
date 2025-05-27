#!/bin/bash

# Read the .secrets/pgadmin file
username=${POSTGRES_USER:-"pgadmin"}
secret=`cat ${POSTGRES_PASSWORD_FILE}`
host='localhost'
port='5432'
database=${MIGRATIONS_DB:-"app"}
sslmode='disable'

migrations=${MIGRATIONS_PATH:-"./migrations"}
conn_str="postgres://$username:$secret@$host:$port/$database?sslmode=$sslmode"

# Perform the PostgreSQL dump
if [ $1 = "down" ]; then
    bash -c "migrate -path $migrations -database $conn_str down"
else
    bash -c "migrate -path $migrations -database $conn_str up"
fi

