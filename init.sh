#!/bin/bash

# Debugging: Check if files exist
echo "Checking if files exist..."
if [ ! -f /docker-entrypoint-initdb.d/init.sql.template ]; then
  echo "init.sql.template not found!"
  exit 1
fi

if [ ! -f /docker-entrypoint-initdb.d/init.sh ]; then
  echo "init.sh not found!"
  exit 1
fi

# Change to the postgres user before writing the init.sql file
echo "Running script as postgres user..."
su - postgres -c "envsubst < /docker-entrypoint-initdb.d/init.sql.template > /docker-entrypoint-initdb.d/init.sql"

