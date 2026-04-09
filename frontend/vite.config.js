import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const wailsConfigPath = resolve(__dirname, '../wails.json')
const wailsConfig = JSON.parse(readFileSync(wailsConfigPath, 'utf-8'))
const appVersion = wailsConfig?.info?.productVersion || 'dev'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
})
