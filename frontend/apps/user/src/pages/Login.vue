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

// 2FA step state
const step = ref<'password' | 'totp' | 'recover'>('password')
const tempToken = ref('')
const totpCode = ref('')
const recoveryCode = ref('')

async function login() {
  error.value = ''
  loading.value = true
  try {
    const result = await auth.login(username.value, password.value)
    if (result.requires_2fa) {
      tempToken.value = result.temp_token
      step.value = 'totp'
    } else {
      router.push('/dashboard')
    }
  } catch {
    error.value = 'Invalid username or password'
  } finally {
    loading.value = false
  }
}

async function verifyTOTP() {
  error.value = ''
  loading.value = true
  try {
    await auth.verifyTOTP(tempToken.value, totpCode.value)
    router.push('/dashboard')
  } catch {
    error.value = 'Invalid authentication code'
  } finally {
    loading.value = false
  }
}

async function recoverTOTP() {
  error.value = ''
  loading.value = true
  try {
    await auth.recoverTOTP(tempToken.value, recoveryCode.value)
    router.push('/dashboard')
  } catch {
    error.value = 'Invalid recovery code'
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

      <!-- Step 1: Password -->
      <form v-if="step === 'password'" @submit.prevent="login" class="space-y-4">
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

      <!-- Step 2: TOTP code -->
      <form v-else-if="step === 'totp'" @submit.prevent="verifyTOTP" class="space-y-4">
        <div class="text-center mb-2">
          <div class="w-10 h-10 bg-indigo-100 rounded-full flex items-center justify-center mx-auto mb-2">
            <svg class="w-5 h-5 text-indigo-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="5" y="11" width="14" height="10" rx="2"/><path d="M12 3a4 4 0 0 1 4 4v4H8V7a4 4 0 0 1 4-4z"/>
            </svg>
          </div>
          <p class="text-sm font-medium text-gray-800">Two-Factor Authentication</p>
          <p class="text-xs text-gray-500 mt-1">Enter the 6-digit code from your authenticator app.</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Authentication Code</label>
          <input v-model="totpCode" type="text" inputmode="numeric" pattern="[0-9]*" maxlength="6"
            placeholder="000000" autofocus required
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm text-center font-mono tracking-widest focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        </div>
        <div v-if="error" class="text-red-500 text-xs text-center">{{ error }}</div>
        <button type="submit" :disabled="loading"
          class="w-full bg-indigo-600 text-white rounded-md py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ loading ? 'Verifying...' : 'Verify' }}
        </button>
        <button type="button" @click="step = 'recover'"
          class="w-full text-xs text-gray-400 hover:text-gray-600 text-center">
          Use a recovery code instead
        </button>
      </form>

      <!-- Step 3: Recovery code -->
      <form v-else-if="step === 'recover'" @submit.prevent="recoverTOTP" class="space-y-4">
        <div class="text-center mb-2">
          <p class="text-sm font-medium text-gray-800">Recovery Code</p>
          <p class="text-xs text-gray-500 mt-1">Enter one of your 8 single-use recovery codes.</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Recovery Code</label>
          <input v-model="recoveryCode" type="text" placeholder="xxxxxxxxxx" required
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        </div>
        <div v-if="error" class="text-red-500 text-xs text-center">{{ error }}</div>
        <button type="submit" :disabled="loading"
          class="w-full bg-indigo-600 text-white rounded-md py-2 text-sm font-medium hover:bg-indigo-700 disabled:opacity-50 transition-colors">
          {{ loading ? 'Verifying...' : 'Recover Account' }}
        </button>
        <button type="button" @click="step = 'totp'"
          class="w-full text-xs text-gray-400 hover:text-gray-600 text-center">
          Back to authenticator code
        </button>
      </form>
    </div>
  </div>
</template>
