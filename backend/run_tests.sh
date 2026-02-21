#!/bin/bash

echo "=== Starting Integration Tests ==="

# Start the server in background
echo "Starting server..."
go run . &
SERVER_PID=$!

# Wait for server to be ready
echo "Waiting for server to be ready..."
sleep 3

# Check if server is responding
curl -s http://localhost:8080/health > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ Server is healthy"
else
    echo "❌ Server is not healthy"
    kill $SERVER_PID
    exit 1
fi

# Run the tests
echo "Running WebSocket tests..."
go test -v -timeout 30s

# Kill the server
echo "Shutting down server..."
kill $SERVER_PID

echo "=== Tests Complete ==="