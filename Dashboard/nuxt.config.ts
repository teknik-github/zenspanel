// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@vueuse/nuxt'
  ],

  devtools: {
    enabled: true
  },

  css: [
    '@xterm/xterm/css/xterm.css',
    '~/assets/css/main.css'
  ],

  runtimeConfig: {
    // Server-only — never exposed to the browser
    backendUrl: 'http://127.0.0.1:8080'
  },

  routeRules: {
    // API proxy — always run on server, never prerender
    '/api/v1/**': { ssr: false },
    // Login page — client-rendered only
    '/admin/login': { ssr: false }
  },

  compatibilityDate: '2024-07-11',

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
