#!/bin/sh
# Wait for service to be available
# Usage: wait-for-service.sh <host> <port> [timeout]

HOST="${1}"
PORT="${2}"
TIMEOUT="${3:-60}"

if [ -z "$HOST" ] || [ -z "$PORT" ]; then
    echo "Usage: $0 <host> <port> [timeout]"
    exit 1
fi

echo "Waiting for $HOST:$PORT to be available..."

retries=0
while [ $retries -lt $TIMEOUT ]; do
    if nc -z "$HOST" "$PORT" 2>/dev/null; then
        echo "$HOST:$PORT is available"
        exit 0
    fi
    retries=$((retries + 1))
    echo "Waiting... ($retries/$TIMEOUT)"
    sleep 1
done

echo "ERROR: $HOST:$PORT is not available after $TIMEOUT seconds"
exit 1
