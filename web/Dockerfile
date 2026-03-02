# BPMN Workflow Platform - Web Frontend Dockerfile

# Build stage
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm install

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM nginx:1.25-alpine

# Copy custom nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Copy built assets from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Add non-root user
RUN adduser -D -u 1000 bpmn && \
    chown -R bpmn:bpmn /usr/share/nginx/html && \
    chown -R bpmn:bpmn /var/cache/nginx && \
    chown -R bpmn:bpmn /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R bpmn:bpmn /var/run/nginx.pid

# Switch to non-root user
USER bpmn

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
