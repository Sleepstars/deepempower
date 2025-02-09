#!/bin/bash

# Build all binaries
echo "Building binaries..."
make build

# Start mock servers
echo "Starting mock servers..."
./bin/mockserver -port 8001 & 
MOCK1_PID=$!
./bin/mockserver -port 8002 &
MOCK2_PID=$!

# Wait for servers to start
sleep 2

# Start main server
echo "Starting main server..."
./bin/deepempower &
MAIN_PID=$!

# Wait for main server to start
sleep 2

# Test the API
echo "Testing API..."
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Normal",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

echo -e "\n\nPress Ctrl+C to stop all servers..."

# Wait for Ctrl+C
trap 'kill $MOCK1_PID $MOCK2_PID $MAIN_PID; exit 0' SIGINT
wait
