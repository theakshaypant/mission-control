import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/items': 'http://localhost:5040',
      '/summary': 'http://localhost:5040',
      '/sync': 'http://localhost:5040',
    },
  },
})
