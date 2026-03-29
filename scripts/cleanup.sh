#!/bin/bash

# cleanup.sh - Script to format and check Go code

echo "Formatting Go code..."
go fmt ./...

echo "Running goimports if available..."
which goimports > /dev/null && goimports -w .

echo "Running golint if available..."
which golint > /dev/null && golint ./...

echo "Running go vet..."
go vet ./...

echo "Running tests..."
go test ./... -short

echo "Checking for unused dependencies..."
go mod tidy

echo "Cleanup complete!"