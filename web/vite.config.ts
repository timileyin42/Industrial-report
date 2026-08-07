import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // Matches DASHBOARD_ORIGINS convention on the Go side — see README's
    // local dev wiring section (export DASHBOARD_ORIGINS=http://localhost:5173).
    port: 5173,
  },
})
