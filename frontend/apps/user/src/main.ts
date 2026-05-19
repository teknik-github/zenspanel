import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth'
import './style.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)

const auth = useAuthStore()

// Admin "Login as User" passes the impersonation token in the URL hash
// as #impersonate=<token>. We read it here before mounting so the SPA
// boots already authenticated as the target user. The hash is never
// sent to the server and is cleared after we consume it.
const hash = window.location.hash
const impersonateMatch = hash.match(/[#&]impersonate=([^&]+)/)
if (impersonateMatch) {
  const token = decodeURIComponent(impersonateMatch[1])
  localStorage.setItem('token', token)
  auth.token = token
  // Strip the hash so it doesn't persist on reload or appear in history.
  history.replaceState(null, '', window.location.pathname + window.location.search)
}

const boot = auth.token ? auth.fetchMe().catch(() => {}) : Promise.resolve()
boot.finally(() => {
  app.use(router)
  app.mount('#app')
})
