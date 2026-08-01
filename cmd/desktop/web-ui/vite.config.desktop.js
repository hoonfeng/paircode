import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { fileURLToPath } from 'url'
import fs from 'fs'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const SHARED_SRC = path.resolve(__dirname, '../../companion/web-ui')

export default defineConfig({
  root: SHARED_SRC,
  plugins: [
    vue(),
    {
      name: 'desktop-inject-libs',
      transformIndexHtml(html) {
        // Inject global library scripts BEFORE the app script.
        // Must be loaded in this order: Vue → Pinia → VueRouter → App
        const libScripts = 
          '<script src="./libs/vue.global.prod.js"></script>\n' +
          '<script src="./libs/pinia.iife.prod.js"></script>\n' +
          '<script src="./libs/vue-router.global.prod.js"></script>\n'
        // Insert before the first <script> tag (the app bundle)
        return html.replace('<script', libScripts + '<script')
      },
      closeBundle() {
        // Copy global library builds to dist/libs/
        const libsDir = path.resolve(__dirname, 'dist', 'libs')
        fs.mkdirSync(libsDir, { recursive: true })
        const moduleDir = path.resolve(__dirname, 'node_modules')
        const files = {
          'vue.global.prod.js': 'vue/dist/vue.global.prod.js',
          'pinia.iife.prod.js': 'pinia/dist/pinia.iife.prod.js',
          'vue-router.global.prod.js': 'vue-router/dist/vue-router.global.prod.js',
        }
        for (const [dest, src] of Object.entries(files)) {
          fs.copyFileSync(path.join(moduleDir, src), path.join(libsDir, dest))
          console.log(`  copied: ${src} → dist/libs/${dest}`)
        }
        // Post-process index.html: wb-ui jsc does NOT support ES modules, so
        // turn the app bundle's <script type="module"> into a plain <script>.
        // The bundle is built as IIFE (see rollupOptions.output.format), so
        // it runs fine as a classic script loaded in order after the libs.
        const indexPath = path.resolve(__dirname, 'dist', 'index.html')
        let html = fs.readFileSync(indexPath, 'utf8')
        html = html.replace('<script type="module" crossorigin', '<script')
        html = html.replace('<script type="module"', '<script')
        fs.writeFileSync(indexPath, html)
        console.log('  post-processed index.html: module script → classic script')
      }
    }
  ],
  base: './',
  build: {
    outDir: path.resolve(__dirname, 'dist'),
    emptyOutDir: true,
    target: 'es2015',
    minify: false,
    assetsInlineLimit: 8192,
    cssCodeSplit: false,
    rollupOptions: {
      external: ['vue', 'pinia', 'vue-router'],
      output: {
        format: 'iife',
        inlineDynamicImports: true,
        globals: {
          vue: 'Vue',
          pinia: 'Pinia',
          'vue-router': 'VueRouter'
        },
        entryFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
      },
    },
  },
})
