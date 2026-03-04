import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: '@wf/tasklist',
      filename: 'remoteEntry.js',
      exposes: {
        './Tasklist': './src/Tasklist',
      },
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
          vendor: ['react', 'react-dom'],
        },
      },
    },
  },
  server: {
    port: 3002,
    strictPort: true,
    cors: true,
  },
  preview: {
    port: 3002,
    strictPort: true,
  },
});
