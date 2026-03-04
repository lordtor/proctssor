#!/bin/bash

# wait-for-service.sh - Ждет пока сервис станет доступным

set -e

host="$1"
port="$2"
timeout="${3:-60}"

if [ -z "$host" ] || [ -z "$port" ]; then
    echo "Usage: $0 <host> <port> [timeout_seconds]"
    exit 1
fi

echo "Waiting for $host:$port to be available..."

start_time=$(date +%s)

while ! nc -z "$host" "$port" 2>/dev/null; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    
    if [ "$elapsed" -ge "$timeout" ]; then
        echo "Timeout waiting for $host:$port after ${timeout}s"
        exit 1
    fi
    
    echo "Waiting... ($elapsed/${timeout}s)"
    sleep 1
done

echo "$host:$port is available!"
exit 0
