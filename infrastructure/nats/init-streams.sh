#!/bin/sh
# NATS JetStream Initialization Script
# Creates streams and consumers for BPMN Workflow Platform

set -e

NATS_URL="${NATS_URL:-http://localhost:4222}"
MAX_RETRIES=30
RETRY_INTERVAL=1

echo "Waiting for NATS to be ready..."
retries=0
while [ $retries -lt $MAX_RETRIES ]; do
    if nats --server "$NATS_URL" server ping 2>/dev/null; then
        echo "NATS is ready"
        break
    fi
    retries=$((retries + 1))
    echo "Waiting for NATS... ($retries/$MAX_RETRIES)"
    sleep $RETRY_INTERVAL
done

if [ $retries -eq $MAX_RETRIES ]; then
    echo "ERROR: NATS is not available after $MAX_RETRIES attempts"
    exit 1
fi

# Add JetStream to the account
echo "Enabling JetStream..."
nats --server "$NATS_URL" account info || nats --server "$NATS_URL" jetserver add

# Import streams configuration
echo "Creating streams..."
if [ -f /nats/streams.json ]; then
    nats --server "$NATS_URL" streams add /nats/streams.json
else
    echo "WARNING: streams.json not found, skipping stream creation"
fi

echo "NATS JetStream initialization complete"
