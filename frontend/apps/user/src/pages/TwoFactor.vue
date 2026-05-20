<script setup lang="ts">
import { ref } from 'vue'
import { authApi } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

// Setup flow state
const step = ref<'idle' | 'setup' | 'confirm' | 'done'>('idle')
const qrUrl = ref('')
const secret = ref('')
const recoveryCodes = ref<string[]>([])
const confirmCode = ref('')
const disableCode = ref('')
const error = ref('')
const loading = ref(false)
const showDisable = ref(false)
const copied = ref(false)

async function startSetup() {
  error.value = ''
  loading.value = true
  try {
    const res = await authApi.twofa.setup()
    qrUrl.value = res.data.qr_url
    secret.value = res.data.secret
    recoveryCodes.value = res.data.recovery_codes
    step.value = 'setup'
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to start 2FA setup'
  } finally {
    loading.value = false
  }
}

async function confirmSetup() {
  error.value = ''
  loading.value = true
  try {
    await authApi.twofa.confirm(confirmCode.value)
    await auth.fetchMe()
    step.value = 'done'
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Invalid code'
  } finally {
    loading.value = false
  }
}

async function disable2FA() {
  error.value = ''
  loading.value = true
  try {
    await authApi.twofa.disable(disableCode.value)
    await auth.fetchMe()
    showDisable.value = false
    disableCode.value = ''
    step.value = 'idle'
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Invalid code'
  } finally {
    loading.value = false
  }
}

function copyRecoveryCodes() {
  navigator.clipboard.writeText(recoveryCodes.value.join('\n'))
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

function downloadRecoveryCodes() {
  const blob = new Blob([recoveryCodes.value.join('\n')], { type: 'text/plain' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = 'zenspanel-recovery-codes.txt'
  a.click()
}
</script>

<template>
  <div class="space-y-4 max-w-lg">
    <h1 class="text-lg font-semibold text-gray-800">Two-Factor Authentication</h1>

    <!-- Current status -->
    <div class="bg-white border border-gray-200 rounded-lg p-4">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-gray-800">Status</h2>
          <p class="text-xs text-gray-500 mt-0.5">
            {{ auth.user?.totp_enabled ? 'Your account is protected with 2FA.' : '2FA is not enabled on your account.' }}
          </p>
        </div>
        <span class="px-2 py-0.5 rounded text-[10px] font-medium"
          :class="auth.user?.totp_enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
          {{ auth.user?.totp_enabled ? 'Enabled' : 'Disabled' }}
        </span>
      </div>

      <div class="mt-3 flex gap-2">
        <button v-if="!auth.user?.totp_enabled && step === 'idle'" @click="startSetup" :disabled="loading"
          class="text-xs bg-indigo-600 text-white px-3 py-1.5 rounded hover:bg-indigo-700 disabled:opacity-50">
          Enable 2FA
        </button>
        <button v-if="auth.user?.totp_enabled" @click="showDisable = true"
          class="text-xs text-red-600 border border-red-200 px-3 py-1.5 rounded hover:bg-red-50">
          Disable 2FA
        </button>
      </div>
    </div>

    <!-- Setup: QR code -->
    <div v-if="step === 'setup'" class="bg-white border border-gray-200 rounded-lg p-4 space-y-4">
      <h2 class="text-sm font-semibold text-gray-800">Step 1 — Scan QR Code</h2>
      <p class="text-xs text-gray-500">Open your authenticator app (Google Authenticator, Authy, etc.) and scan this QR code.</p>

      <div class="flex justify-center">
        <img :src="qrUrl" alt="TOTP QR Code" class="w-48 h-48 border border-gray-200 rounded" />
      </div>

      <div class="bg-gray-50 rounded p-2 text-center">
        <p class="text-[10px] text-gray-400 mb-1">Or enter this key manually:</p>
        <code class="text-xs font-mono text-gray-700 break-all">{{ secret }}</code>
      </div>

      <h2 class="text-sm font-semibold text-gray-800 pt-2">Step 2 — Save Recovery Codes</h2>
      <p class="text-xs text-gray-500">Save these 8 single-use recovery codes in a safe place. Each can only be used once.</p>

      <div class="bg-gray-900 rounded p-3 font-mono text-xs text-green-400 grid grid-cols-2 gap-1">
        <span v-for="code in recoveryCodes" :key="code">{{ code }}</span>
      </div>

      <div class="flex gap-2">
        <button @click="copyRecoveryCodes"
          class="flex-1 text-xs border border-gray-200 px-3 py-1.5 rounded hover:bg-gray-50">
          {{ copied ? 'Copied!' : 'Copy' }}
        </button>
        <button @click="downloadRecoveryCodes"
          class="flex-1 text-xs border border-gray-200 px-3 py-1.5 rounded hover:bg-gray-50">
          Download
        </button>
      </div>

      <h2 class="text-sm font-semibold text-gray-800 pt-2">Step 3 — Confirm</h2>
      <p class="text-xs text-gray-500">Enter the 6-digit code from your authenticator app to activate 2FA.</p>

      <div class="flex gap-2">
        <input v-model="confirmCode" type="text" inputmode="numeric" maxlength="6" placeholder="000000"
          class="flex-1 border border-gray-200 rounded px-3 py-2 text-sm font-mono text-center tracking-widest focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <button @click="confirmSetup" :disabled="loading || confirmCode.length < 6"
          class="text-xs bg-indigo-600 text-white px-4 py-2 rounded hover:bg-indigo-700 disabled:opacity-50">
          {{ loading ? '...' : 'Activate' }}
        </button>
      </div>
      <p v-if="error" class="text-xs text-red-600">{{ error }}</p>
    </div>

    <!-- Done -->
    <div v-if="step === 'done'" class="bg-green-50 border border-green-200 rounded-lg p-4 text-center">
      <svg class="w-8 h-8 text-green-500 mx-auto mb-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>
      </svg>
      <p class="text-sm font-semibold text-green-800">2FA Enabled</p>
      <p class="text-xs text-green-600 mt-1">Your account is now protected. Keep your recovery codes safe.</p>
    </div>

    <!-- Disable modal -->
    <div v-if="showDisable" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Disable 2FA</h2>
        <p class="text-sm text-gray-500 mb-4">Enter your current authenticator code to confirm.</p>
        <input v-model="disableCode" type="text" inputmode="numeric" maxlength="6" placeholder="000000"
          class="w-full border border-gray-200 rounded px-3 py-2 text-sm font-mono text-center tracking-widest focus:outline-none focus:ring-2 focus:ring-indigo-500 mb-3" />
        <p v-if="error" class="text-xs text-red-600 mb-2">{{ error }}</p>
        <div class="flex gap-2">
          <button @click="showDisable = false; error = ''"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="disable2FA" :disabled="loading || disableCode.length < 6"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700 disabled:opacity-50">
            {{ loading ? '...' : 'Disable' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
