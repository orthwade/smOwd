#!/bin/bash

# Check if the database already exists
echo "Checking if database exists"
psql -U ${DB_SUPERUSER} -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" | grep -q 1 || {
  # If database does not exist, create it
  echo "Creating database ${DB_NAME}"
  psql -U ${DB_SUPERUSER} -d postgres -c "CREATE DATABASE ${DB_NAME}"
}

# Create the user if it doesn't exist
echo "Checking if user exists"
psql -U ${DB_SUPERUSER} -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" | grep -q 1 || {
  echo "Creating user ${DB_USER}"
  psql -U ${DB_SUPERUSER} -d postgres -c "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}'"
  psql -U ${DB_SUPERUSER} -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER}"
}

# Optionally, you can run additional SQL queries if required
# You can put your SQL file in here or run any needed query
# Example (uncomment the following lines to run init.sql.template or other SQL commands):

# psql -U ${DB_SUPERUSER} -d ${DB_NAME} -f /docker-entrypoint-initdb.d/init.sql.template
