#!/bin/bash

echo "DB_SUPERUSER=${DB_SUPERUSER}"
echo "DB_NAME=${DB_NAME}"
echo "DB_USER=${DB_USER}"

# Check if the database exists
echo "Checking if database exists"
psql -U ${DB_SUPERUSER} -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" | grep -q 1 || {
  # If database does not exist, create it
  echo "Creating database ${DB_NAME}"
  psql -U ${DB_SUPERUSER} -d postgres -c "CREATE DATABASE ${DB_NAME}"
}

# Check if the user exists
echo "Checking if user exists"
psql -U ${DB_SUPERUSER} -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" | grep -q 1 || {
  echo "Creating user ${DB_USER}"
  psql -U ${DB_SUPERUSER} -d postgres -c "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}'"
  psql -U ${DB_SUPERUSER} -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER}"
}
