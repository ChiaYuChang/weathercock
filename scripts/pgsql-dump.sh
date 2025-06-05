#!/bin/bash

# Read the .secrets/pgadmin file
username=${POSTGRES_USER:-"pgadmin"}
secret=`cat ${POSTGRES_PASSWORD_FILE}`
host='localhost'
port='5432'
database=${POSTGRES_APP_DB:-"app"}
sslmode='disable'
conn_str="postgres://$username:$secret@$host:$port/$database?sslmode=$sslmode"
output='./schema.sql'

# Perform the PostgreSQL dump
bash -c "pg_dump postgres://$username:$secret@$host:$port/$database?sslmode=$sslmode -f $output --schema-only"

