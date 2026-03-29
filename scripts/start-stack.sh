#!/bin/bash

set -e

if docker compose version >/dev/null 2>&1; then
	COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
	COMPOSE_CMD="docker-compose"
else
	echo "Error: Neither 'docker compose' nor 'docker-compose' is available."
	exit 1
fi

# Start the database first
echo "Starting database..."
$COMPOSE_CMD -f docker-compose-db.yml up -d

# Give the database some time to initialize
echo "Waiting for database to initialize (10 seconds)..."
sleep 10

# Start the PiPiMink service
echo "Starting PiPiMink service..."
$COMPOSE_CMD -f docker-compose-app.yml up -d

# Give PiPiMink time to start up
echo "Waiting for PiPiMink service to start (5 seconds)..."
sleep 5

echo "All services started!"
echo "PiPiMink is available at: http://localhost:8080"
echo "Swagger UI is available at: http://localhost:8080/swagger/index.html"
echo "pgAdmin is available at: http://localhost:5050"
echo ""
echo "To update PiPiMink models (needed for model selection):"
echo "  ./scripts/update_models.sh"