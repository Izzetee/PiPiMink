#!/bin/bash

set -e

if [ -f .env ]; then
  # shellcheck disable=SC1091
  source .env
fi

# This script tests the PiPiMink's model selection functionality

# Define the API endpoint
API_ENDPOINT="http://localhost:8080/models/update"
# API key for the /models/update endpoint.
# Set ADMIN_API_KEY in your .env file (or export it) — this must match the ADMIN_API_KEY
# value the server was started with.
# The fallback "admin-key-12345" is a local development placeholder only;
# never use it in a networked or production environment.
API_KEY="${ADMIN_API_KEY:-admin-key-12345}"

curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  $API_ENDPOINT

# List all Models
# curl http://localhost:8080/models