import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: '@wf/modeler',
      filename: 'remoteEntry.js',
      exposes: {
        './Modeler': './src/Modeler',
      },
      shared: {
        react: { singleton: true, requiredVersion: '^18.2.0', eager: true },
        'react-dom': { singleton: true, requiredVersion: '^18.2.0', eager: true },
        'react-router-dom': { singleton: true, requiredVersion: '^6.20.0' },
        axios: { singleton: true, requiredVersion: '^1.6.0' },
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
    port: 3001,
    strictPort: true,
    cors: true,
  },
  preview: {
    port: 3001,
    strictPort: true,
  },
});
