#!/bin/bash

echo "Testing WebSocket Server..."

# Test 1: Check if server is running
curl -s http://localhost:8080 > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ Server is running on port 8080"
else
    echo "❌ Server is not running. Start it with: go run ."
    exit 1
fi

# Test 2: Check database connection
psql -h localhost -U editor_user -d collab_editor -c "SELECT 1" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Database connection successful"
else
    echo "❌ Database connection failed. Check PostgreSQL"
fi

# Test 3: Run Go tests
echo "Running WebSocket tests..."
go test -v -run TestWebSocketConnection

echo "Done!"