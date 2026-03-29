#!/bin/bash

# This script tests the PiPiMink's model selection functionality

API_ENDPOINT="http://localhost:8080/chat"
OPENAI_ENDPOINT="http://localhost:8080/v1/chat/completions"

echo "=== Single-turn: technical question ==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "Explain quantum computing principles and their application in cryptography, with code examples in Python"}' \
  $API_ENDPOINT | jq .

echo ""
echo "=== Single-turn: creative writing ==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "Write a short poem about artificial intelligence"}' \
  $API_ENDPOINT | jq .

echo ""
echo "=== Single-turn: mathematical problem ==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "Solve the differential equation dy/dx = 2xy with the initial condition y(0) = 1"}' \
  $API_ENDPOINT | jq .

echo ""
echo "=== Multi-turn: conversation history (native /chat) ==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user",      "content": "How do I implement a binary search tree in Go?"},
      {"role": "assistant", "content": "A binary search tree in Go can be implemented with a Node struct containing a value and left/right pointers..."},
      {"role": "user",      "content": "Can you now show me how to add a delete operation to that implementation?"}
    ]
  }' \
  $API_ENDPOINT | jq .

echo ""
echo "=== Multi-turn: conversation history (OpenAI-compatible /v1/chat/completions) ==="
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4-turbo",
    "messages": [
      {"role": "user",      "content": "What is the difference between a mutex and a semaphore?"},
      {"role": "assistant", "content": "A mutex provides mutual exclusion for a single resource, while a semaphore controls access to a pool of resources..."},
      {"role": "user",      "content": "Give me a concrete Go example showing both."}
    ]
  }' \
  $OPENAI_ENDPOINT | jq .
