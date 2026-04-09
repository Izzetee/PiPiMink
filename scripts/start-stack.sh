#!/bin/bash

set -e

# Always run from the project root, regardless of where the script is invoked from
cd "$(dirname "$0")/.."

if docker compose version >/dev/null 2>&1; then
	COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
	COMPOSE_CMD="docker-compose"
else
	echo "Error: Neither 'docker compose' nor 'docker-compose' is available."
	exit 1
fi

# Parse flags
WITH_AUTHENTIK=true
for arg in "$@"; do
	case $arg in
		--with-authentik)
			WITH_AUTHENTIK=true
			;;
	esac
done

# Start the database first
echo "Starting database..."
$COMPOSE_CMD -f docker-compose-db.yml up -d

# Give the database some time to initialize
echo "Waiting for database to initialize (10 seconds)..."
sleep 10

# Optionally start Authentik identity provider
if [ "$WITH_AUTHENTIK" = true ]; then
	echo "Starting Authentik identity provider..."
	$COMPOSE_CMD -f docker-compose-authentik.yml up -d

	echo "Waiting for Authentik to initialize (15 seconds)..."
	sleep 15
fi

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

if [ "$WITH_AUTHENTIK" = true ]; then
	echo "Authentik is available at: http://localhost:9000"
	echo ""
	echo "First-time Authentik setup:"
	echo "  1. Open http://localhost:9000/if/flow/initial-setup/"
	echo "  2. Create an admin account"
	echo "  3. Create an OAuth2/OpenID provider + application for PiPiMink"
	echo "  4. Set redirect URI to http://localhost:8080/auth/callback"
	echo "  5. Copy Client ID / Secret into your .env (OAUTH_CLIENT_ID, OAUTH_CLIENT_SECRET)"
fi

echo ""
echo "To update PiPiMink models (needed for model selection):"
echo "  ./scripts/update_models.sh"
