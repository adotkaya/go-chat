#!/bin/sh
set -e

echo "Starting Go Chat Application..."

# Wait for database to be ready
echo "Waiting for database to be ready..."
until nc -z -v -w30 db 5432; do
  echo "Waiting for database connection..."
  sleep 1
done

echo "Database is ready!"

# Run migrations
echo "Running database migrations..."
/app/migrate up

if [ $? -eq 0 ]; then
  echo "Migrations completed successfully!"
else
  echo "Migration failed!"
  exit 1
fi

# Execute the main application
echo "Starting application..."
exec "$@"
