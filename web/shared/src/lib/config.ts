/**
 * Application configuration
 * All environment variables are accessed via import.meta.env
 */

export const config = {
  // API Configuration
  api: {
    baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1',
    wsUrl: import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1',
  },

  // Micro-frontends URLs (development)
  microfrontends: {
    modeler: import.meta.env.VITE_MODELER_URL || 'http://localhost:3001',
    tasklist: import.meta.env.VITE_TASKLIST_URL || 'http://localhost:3002',
    monitor: import.meta.env.VITE_MONITOR_URL || 'http://localhost:3003',
  },

  // Authentication
  auth: {
    provider: import.meta.env.VITE_AUTH_PROVIDER || 'local',
    keycloak: {
      url: import.meta.env.VITE_KEYCLOAK_URL,
      realm: import.meta.env.VITE_KEYCLOAK_REALM,
      clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID,
    },
  },

  // Feature Flags
  features: {
    sse: import.meta.env.VITE_ENABLE_SSE === 'true',
    websocket: import.meta.env.VITE_ENABLE_WEBSOCKET === 'true',
    analytics: import.meta.env.VITE_ENABLE_ANALYTICS === 'true',
  },

  // App Info
  version: import.meta.env.VITE_VERSION || '2.0.0',
  environment: import.meta.env.VITE_BUILD_ENV || 'development',
};

// Helper to get WebSocket URL for instance monitoring
export const getWebSocketUrl = (instanceId: string): string => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  return `${protocol}//${host}/api/v1/ws/instances/${instanceId}`;
};

// Helper to get SSE URL for task notifications
export const getSSEUrl = (assignee: string): string => {
  const baseUrl = config.api.baseUrl;
  return `${baseUrl}/sse/tasks?assignee=${encodeURIComponent(assignee)}`;
};

export default config;
