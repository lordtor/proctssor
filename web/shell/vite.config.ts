import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';

// Определяем окружение
const isProduction = process.env.NODE_ENV === 'production';

// В production используем относительные пути (все микрофронтенды на одном домене через nginx)
// В development используем прямые ссылки на порты
const remotes = isProduction
  ? {
      '@wf/modeler': '/modeler/assets/remoteEntry.js',
      '@wf/tasklist': '/tasklist/assets/remoteEntry.js',
      '@wf/monitor': '/monitor/assets/remoteEntry.js',
    }
  : {
      '@wf/modeler': 'http://localhost:3001/assets/remoteEntry.js',
      '@wf/tasklist': 'http://localhost:3002/assets/remoteEntry.js',
      '@wf/monitor': 'http://localhost:3003/assets/remoteEntry.js',
    };

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: '@wf/shell',
      remotes,
      shared: {
        react: { singleton: true, requiredVersion: '^18.2.0', eager: true },
        'react-dom': { singleton: true, requiredVersion: '^18.2.0', eager: true },
        'react-router-dom': { singleton: true, requiredVersion: '^6.20.0' },
        axios: { singleton: true, requiredVersion: '^1.6.0' },
        zustand: { singleton: true, requiredVersion: '^4.4.0' },
      },
    }),
  ],
  build: {
    target: 'esnext',
    minify: true,
    cssMinify: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
        },
      },
    },
  },
  server: {
    port: 3000,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
  preview: {
    port: 3000,
    strictPort: true,
  },
});
