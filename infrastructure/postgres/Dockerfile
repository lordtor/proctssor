# PostgreSQL for BPMN Workflow Platform
FROM postgres:15-bookworm

# Install pg_cron extension from PostgreSQL PGDG repository
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    gnupg \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/postgresql.gpg] http://apt.postgresql.org/pub/repos/apt bookworm-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y postgresql-15-cron \
    && rm -rf /var/lib/apt/lists/*

# Create migrations directory
RUN mkdir -p /migrations

# Set working directory
WORKDIR /

# Expose PostgreSQL port
EXPOSE 5432

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=5 \
    CMD pg_isready -U bpmn || exit 1
