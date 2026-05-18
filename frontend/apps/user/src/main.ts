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
const boot = auth.token ? auth.fetchMe().catch(() => {}) : Promise.resolve()
boot.finally(() => {
  app.use(router)
  app.mount('#app')
})
