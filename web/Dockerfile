# BPMN Workflow Platform - Web Frontend Multi-stage Dockerfile
# Builds all micro-frontends with Module Federation support

# Stage 1: Build Shell (Host)
FROM node:20-alpine AS shell-builder

WORKDIR /app/shell
COPY shell/package*.json ./
RUN npm install
COPY shell/ ./
RUN npm run build

# Stage 2: Build Modeler
FROM node:20-alpine AS modeler-builder

WORKDIR /app/modeler
COPY modeler/package*.json ./
RUN npm install
COPY modeler/ ./
RUN npm run build

# Stage 3: Build Tasklist
FROM node:20-alpine AS tasklist-builder

WORKDIR /app/tasklist
COPY tasklist/package*.json ./
RUN npm install
COPY tasklist/ ./
RUN npm run build

# Stage 4: Build Monitor
FROM node:20-alpine AS monitor-builder

WORKDIR /app/monitor
COPY monitor/package*.json ./
RUN npm install
COPY monitor/ ./
RUN npm run build

# Stage 5: Production with Nginx
FROM nginx:1.25-alpine

# Copy nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Copy built assets from all builders
COPY --from=shell-builder /app/shell/dist /usr/share/nginx/html/shell
COPY --from=modeler-builder /app/modeler/dist /usr/share/nginx/html/modeler
COPY --from=tasklist-builder /app/tasklist/dist /usr/share/nginx/html/tasklist
COPY --from=monitor-builder /app/monitor/dist /usr/share/nginx/html/monitor

# Add non-root user
RUN adduser -D -u 1000 bpmn && \
    chown -R bpmn:bpmn /usr/share/nginx/html && \
    chown -R bpmn:bpmn /var/cache/nginx && \
    chown -R bpmn:bpmn /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R bpmn:bpmn /var/run/nginx.pid

# Switch to non-root user
USER bpmn

# Expose ports for each micro-frontend
EXPOSE 3000 3001 3002 3003

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/shell/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
