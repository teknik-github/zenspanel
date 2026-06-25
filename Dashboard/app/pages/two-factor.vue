<script setup lang="ts">
type Setup2FAResponse = {
  secret: string
  qr_url?: string
  recovery_codes: string[]
}

type ApiError = {
  data?: {
    error?: string
  }
}

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Request failed.'
}

const toast = useToast()
const setupData = ref<Setup2FAResponse | null>(null)
const code = ref('')
const loading = ref(false)

async function setup2FA() {
  loading.value = true
  try {
    setupData.value = await $fetch<Setup2FAResponse>('/api/v1/auth/2fa/setup', { method: 'POST' })
    toast.add({ title: 'Setup started', description: 'Scan QR code with authenticator app', color: 'success' })
  } catch (error: unknown) {
    toast.add({ title: 'Error', description: getErrorMessage(error), color: 'error' })
  } finally {
    loading.value = false
  }
}

async function confirm2FA() {
  if (!code.value) return
  loading.value = true
  try {
    await $fetch('/api/v1/auth/2fa/confirm', { method: 'POST', body: { code: code.value } })
    toast.add({ title: '2FA Enabled', description: 'Your account is now protected.', color: 'success' })
    setupData.value = null
    code.value = ''
  } catch (error: unknown) {
    toast.add({ title: 'Error', description: getErrorMessage(error), color: 'error' })
  } finally {
    loading.value = false
  }
}

const disableCode = ref('')
const disableLoading = ref(false)
async function disable2FA() {
  disableLoading.value = true
  try {
    await $fetch('/api/v1/auth/2fa', { method: 'DELETE', body: { code: disableCode.value } })
    toast.add({ title: '2FA Disabled', color: 'success' })
    disableCode.value = ''
  } catch (error: unknown) {
    toast.add({ title: 'Error', description: getErrorMessage(error), color: 'error' })
  } finally {
    disableLoading.value = false
  }
}

const auth = useAuth()
const has2FA = computed(() => auth.user.value?.totp_enabled)
</script>

<template>
  <UDashboardPanel id="two-factor">
    <template #header>
      <UDashboardNavbar title="Two-Factor Authentication">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="min-h-[calc(100dvh-9rem)] flex items-center justify-center">
        <div class="w-full max-w-2xl space-y-6">
          <UCard v-if="has2FA">
            <template #header>
              <h3 class="font-semibold text-success">
                2FA is Active
              </h3>
            </template>

            <div class="space-y-4">
              <p class="text-sm text-dimmed">
                Two-factor authentication is protecting your account.
              </p>
              <UFormField label="Enter current code to disable">
                <div class="flex flex-col sm:flex-row sm:items-center gap-3">
                  <UInput
                    v-model="disableCode"
                    placeholder="000000"
                    :disabled="disableLoading"
                    class="w-full"
                  />
                  <UButton
                    label="Disable 2FA"
                    color="error"
                    :loading="disableLoading"
                    class="justify-center"
                    @click="disable2FA"
                  />
                </div>
              </UFormField>
            </div>
          </UCard>

          <UCard v-else-if="!setupData">
            <template #header>
              <h3 class="font-semibold">
                Setup Two-Factor Auth
              </h3>
            </template>

            <div class="text-center sm:text-left">
              <p class="text-sm text-dimmed mb-4">
                Add an extra layer of security with TOTP-based 2FA using your authenticator app.
              </p>
              <UButton
                label="Setup 2FA"
                icon="i-lucide-fingerprint"
                :loading="loading"
                class="justify-center"
                @click="setup2FA"
              />
            </div>
          </UCard>

          <UCard v-else>
            <template #header>
              <h3 class="font-semibold">
                Confirm Setup
              </h3>
            </template>

            <div class="space-y-4">
              <div class="p-3 bg-elevated rounded-lg overflow-x-auto">
                <p class="text-xs text-dimmed mb-1">
                  Secret (manual entry)
                </p>
                <code class="text-base sm:text-lg font-mono font-bold break-all">{{ setupData.secret }}</code>
              </div>
              <div v-if="setupData.qr_url" class="text-center">
                <img
                  :src="setupData.qr_url"
                  alt="QR Code"
                  class="inline-block rounded-lg bg-white p-2"
                  width="180"
                  height="180"
                >
              </div>
              <div class="p-3 bg-warning/10 border border-warning/20 rounded-lg">
                <p class="text-sm font-semibold text-warning">
                  Save your recovery codes
                </p>
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-1 mt-2">
                  <code
                    v-for="rc in setupData.recovery_codes"
                    :key="rc"
                    class="text-xs font-mono bg-elevated px-2 py-1 rounded break-all"
                  >
                    {{ rc }}
                  </code>
                </div>
                <p class="text-xs text-dimmed mt-2">
                  These are shown only once. Store them securely.
                </p>
              </div>
              <UFormField label="Enter code from app to confirm">
                <div class="flex flex-col sm:flex-row sm:items-center gap-3">
                  <UInput
                    v-model="code"
                    placeholder="000000"
                    :disabled="loading"
                    class="w-full"
                  />
                  <UButton
                    label="Confirm"
                    color="primary"
                    :loading="loading"
                    class="justify-center"
                    @click="confirm2FA"
                  />
                </div>
              </UFormField>
            </div>
          </UCard>
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
