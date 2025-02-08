#!/bin/bash

# Debugging: Print environment variables to check if they are being loaded correctly
echo "DB_SUPERUSER=${DB_SUPERUSER}"
echo "DB_NAME=${DB_NAME}"
echo "DB_USER=${DB_USER}"
echo "DB_PASSWORD=${DB_PASSWORD}"

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

# Grant user permission to access the public schema and create tables
echo "Granting permissions to user ${DB_USER} on public schema"
psql -U ${DB_SUPERUSER} -d ${DB_NAME} -c "GRANT ALL PRIVILEGES ON SCHEMA public TO ${DB_USER}"
psql -U ${DB_SUPERUSER} -d ${DB_NAME} -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER}"

# You may also want to grant additional privileges (if needed)
# psql -U ${DB_SUPERUSER} -d ${DB_NAME} -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ${DB_USER}"
