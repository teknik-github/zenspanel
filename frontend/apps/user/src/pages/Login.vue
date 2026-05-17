<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function login() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    router.push('/dashboard')
  } catch {
    error.value = 'Invalid username or password'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center">
    <div class="bg-white border border-gray-200 rounded-xl p-8 w-full max-w-sm shadow-sm">
      <div class="flex items-center gap-2 text-indigo-600 font-bold text-xl mb-1">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
        </svg>
        ZensPanel
      </div>
      <p class="text-gray-400 text-sm mb-6">User Panel</p>

      <form @submit.prevent="login" class="space-y-4">
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Username</label>
          <input v-model="username" type="text" required
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Password</label>
          <input v-model="password" type="password" required
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
        </div>
        <div v-if="error" class="text-red-500 text-xs">{{ error }}</div>
        <button type="submit" :disabled="loading"
          class="w-full bg-indigo-600 text-white rounded-md py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
      </form>
    </div>
  </div>
</template>
