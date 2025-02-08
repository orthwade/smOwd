#!/bin/bash
set -e

# Wait for PostgreSQL to be ready
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER"; do
  echo "Waiting for postgres to be ready..."
  sleep 2
done

# Create the database if it doesn't exist
echo "Checking if database ${DB_NAME} exists"
DB_EXISTS=$(psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'")
if [ "$DB_EXISTS" != "1" ]; then
  echo "Creating database ${DB_NAME}"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
      CREATE DATABASE ${DB_NAME};
      GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
  EOSQL
else
  echo "Database ${DB_NAME} already exists"
fi

# Create the user if it doesn't exist
echo "Checking if user ${DB_USER} exists"
USER_EXISTS=$(psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'")
if [ "$USER_EXISTS" != "1" ]; then
  echo "Creating user ${DB_USER}"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
      CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';
      GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
  EOSQL
else
  echo "User ${DB_USER} already exists"
fi
