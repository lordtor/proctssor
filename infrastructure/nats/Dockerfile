# NATS for BPMN Workflow Platform
FROM nats:2.10-alpine

# Copy configuration files
COPY nats.conf /nats/nats.conf

# Expose ports
# 4222 - client connections
# 6222 - cluster port (if needed)
# 8222 - HTTP monitoring
EXPOSE 4222 6222 8222

# Run NATS with config
CMD ["nats-server", "-c", "/nats/nats.conf"]
