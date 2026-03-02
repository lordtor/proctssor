# PostgreSQL for BPMN Workflow Platform
FROM postgres:16-alpine

# Install additional tools for migrations
RUN apk add --no-cache \
    bash \
    curl \
    postgresql-contrib

# Create migrations directory
RUN mkdir -p /migrations

# Set working directory
WORKDIR /

# Expose PostgreSQL port
EXPOSE 5432

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=5 \
    CMD pg_isready -U bpmn || exit 1
