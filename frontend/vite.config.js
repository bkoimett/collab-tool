import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/ws': {
        target: 'wss://collab-tool-backend-jjqu.onrender.com',
        ws: true,
      },
      '/api': {
        target: 'https://collab-tool-backend-jjqu.onrender.com',
        changeOrigin: true,
      },
    },
  },
})
