import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  base: '/console/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
  },
  server: {
    port: 5173,
    proxy: {
      '/models': 'http://localhost:8080',
      '/admin': 'http://localhost:8080',
      '/v1': 'http://localhost:8080',
      '/api': 'http://localhost:8080',
      '/chat': 'http://localhost:8080',
      '/benchmarks': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080',
      '/swagger': 'http://localhost:8080',
      '/assets': 'http://localhost:8080',
    },
  },
})
