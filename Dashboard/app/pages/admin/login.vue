<script setup lang="ts">
definePageMeta({
  layout: false
})

const auth = useAuth()
const router = useRouter()

const username = ref('')
const password = ref('')
const totpCode = ref('')
const recoveryCode = ref('')
const showRecovery = ref(false)
const showPassword = ref(false)

if (auth.isAuthenticated.value) {
  await navigateTo('/dashboard')
}

async function handleLogin() {
  try {
    await auth.login(username.value, password.value)
    if (!auth.requires2FA.value) {
      await navigateTo('/dashboard')
    }
  } catch {
    // Error is set in auth.error
  }
}

async function handleVerify2FA() {
  try {
    await auth.verify2FA(totpCode.value)
    await navigateTo('/dashboard')
  } catch {
    // Error is set in auth.error
  }
}

async function handleRecover2FA() {
  try {
    await auth.recover2FA(recoveryCode.value)
    await navigateTo('/dashboard')
  } catch {
    // Error is set in auth.error
  }
}

function toggleRecovery() {
  showRecovery.value = !showRecovery.value
  auth.clearError()
}

watchEffect(() => {
  if (auth.isAuthenticated.value) {
    router.push('/dashboard')
  }
})
</script>

<template>
  <main class="admin-login-root">
    <div class="admin-login-grid" />

    <section class="admin-login-shell">
      <aside class="admin-login-visual" aria-hidden="true">
        <div class="admin-login-mark">
          <div class="admin-login-mark-icon">
            <UIcon name="i-lucide-shield-check" class="size-7" />
          </div>
          <div>
            <p class="admin-login-eyebrow">
              Admin Access
            </p>
            <h2>Control plane</h2>
          </div>
        </div>

        <div class="admin-login-command">
          <div class="admin-login-command-head">
            <span />
            <span />
            <span />
          </div>
          <div class="admin-login-command-row">
            <span class="prompt">$</span>
            <span>audit session --role admin</span>
          </div>
          <div class="admin-login-command-row muted">
            <span class="prompt">></span>
            <span>policy checks ready</span>
          </div>
          <div class="admin-login-command-row success">
            <span class="prompt">></span>
            <span>secure channel established</span>
          </div>
        </div>

        <div class="admin-login-stats">
          <div>
            <span>Roles</span>
            <strong>Scoped</strong>
          </div>
          <div>
            <span>Audit</span>
            <strong>On</strong>
          </div>
          <div>
            <span>2FA</span>
            <strong>Ready</strong>
          </div>
        </div>
      </aside>

      <section class="admin-login-panel">
        <div class="admin-login-card">
          <div class="admin-login-brand">
            <div class="admin-login-icon">
              <UIcon :name="auth.requires2FA.value ? 'i-lucide-shield-check' : 'i-lucide-server'" class="size-6" />
            </div>
            <h1 class="admin-login-title">
              {{ auth.requires2FA.value ? 'Verify Identity' : 'ZensPanel' }}
            </h1>
            <p class="admin-login-desc">
              {{ auth.requires2FA.value ? 'Complete the second authentication step.' : 'Sign in to the administrator console.' }}
            </p>
          </div>

          <template v-if="auth.requires2FA.value">
            <form v-if="!showRecovery" class="admin-login-form" @submit.prevent="handleVerify2FA">
              <div class="admin-login-field">
                <label class="admin-login-label">Authentication Code</label>
                <input
                  v-model="totpCode"
                  type="text"
                  inputmode="numeric"
                  autocomplete="one-time-code"
                  placeholder="000000"
                  maxlength="6"
                  required
                  :disabled="auth.loading.value"
                  class="admin-login-input admin-login-input--totp"
                >
              </div>

              <div v-if="auth.error.value" class="admin-login-error">
                <UIcon name="i-lucide-circle-alert" class="size-4 shrink-0" />
                {{ auth.error.value }}
              </div>

              <button type="submit" :disabled="auth.loading.value" class="admin-login-btn">
                <UIcon v-if="auth.loading.value" name="i-lucide-loader-circle" class="size-4 admin-login-spin" />
                <UIcon v-else name="i-lucide-check" class="size-4" />
                <span>{{ auth.loading.value ? 'Verifying...' : 'Verify Code' }}</span>
              </button>

              <button type="button" class="admin-login-link" @click="toggleRecovery">
                Use recovery code instead
              </button>
            </form>

            <form v-else class="admin-login-form" @submit.prevent="handleRecover2FA">
              <div class="admin-login-field">
                <label class="admin-login-label">Recovery Code</label>
                <input
                  v-model="recoveryCode"
                  type="text"
                  placeholder="XXXX-XXXX-XXXX-XXXX"
                  required
                  :disabled="auth.loading.value"
                  class="admin-login-input"
                >
              </div>

              <div v-if="auth.error.value" class="admin-login-error">
                <UIcon name="i-lucide-circle-alert" class="size-4 shrink-0" />
                {{ auth.error.value }}
              </div>

              <button type="submit" :disabled="auth.loading.value" class="admin-login-btn">
                <UIcon v-if="auth.loading.value" name="i-lucide-loader-circle" class="size-4 admin-login-spin" />
                <UIcon v-else name="i-lucide-key-round" class="size-4" />
                <span>{{ auth.loading.value ? 'Recovering...' : 'Recover Access' }}</span>
              </button>

              <button type="button" class="admin-login-link" @click="toggleRecovery">
                Back to verification code
              </button>
            </form>
          </template>

          <form v-else class="admin-login-form" @submit.prevent="handleLogin">
            <div class="admin-login-field">
              <label class="admin-login-label" for="admin-login-username">Username</label>
              <div class="admin-login-input-wrap">
                <UIcon name="i-lucide-user" class="size-4 admin-login-input-icon" />
                <input
                  id="admin-login-username"
                  v-model="username"
                  type="text"
                  autocomplete="username"
                  placeholder="root"
                  required
                  :disabled="auth.loading.value"
                  class="admin-login-input admin-login-input--with-icon"
                >
              </div>
            </div>

            <div class="admin-login-field">
              <label class="admin-login-label" for="admin-login-password">Password</label>
              <div class="admin-login-input-wrap">
                <UIcon name="i-lucide-lock-keyhole" class="size-4 admin-login-input-icon" />
                <input
                  id="admin-login-password"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                  placeholder="password"
                  required
                  :disabled="auth.loading.value"
                  class="admin-login-input admin-login-input--with-icon admin-login-input--password"
                >
                <button
                  type="button"
                  class="admin-login-eye"
                  :aria-label="showPassword ? 'Hide password' : 'Show password'"
                  @click="showPassword = !showPassword"
                >
                  <UIcon :name="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="size-4" />
                </button>
              </div>
            </div>

            <div v-if="auth.error.value" class="admin-login-error">
              <UIcon name="i-lucide-circle-alert" class="size-4 shrink-0" />
              {{ auth.error.value }}
            </div>

            <button type="submit" :disabled="auth.loading.value" class="admin-login-btn">
              <UIcon v-if="auth.loading.value" name="i-lucide-loader-circle" class="size-4 admin-login-spin" />
              <UIcon v-else name="i-lucide-log-in" class="size-4" />
              <span>{{ auth.loading.value ? 'Authenticating...' : 'Sign In' }}</span>
            </button>
          </form>

          <div class="admin-login-meta">
            <span>ZensPanel v2.0.0</span>
            <span>/</span>
            <span>Admin Console</span>
          </div>
        </div>
      </section>
    </section>
  </main>
</template>

<style scoped>
.admin-login-root { position: relative; min-height: 100vh; display: grid; place-items: center; padding: 2rem; overflow: hidden; background: linear-gradient(135deg, #090a0f 0%, #121826 45%, #241414 100%); font-family: 'Public Sans', ui-sans-serif, system-ui, sans-serif; }
.admin-login-root::before { content: ""; position: absolute; inset: 0; background: linear-gradient(120deg, rgba(248,113,113,0.15), transparent 36%), linear-gradient(300deg, rgba(34,211,238,0.12), transparent 44%); }
.admin-login-grid { position: absolute; inset: 0; background-image: linear-gradient(rgba(255,255,255,0.04) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.04) 1px, transparent 1px); background-size: 44px 44px; mask-image: linear-gradient(to bottom, transparent, black 16%, black 82%, transparent); }
.admin-login-shell { position: relative; z-index: 1; display: grid; grid-template-columns: minmax(0, 1fr) 430px; width: min(100%, 1000px); min-height: 620px; overflow: hidden; border: 1px solid rgba(255,255,255,0.12); border-radius: 24px; background: rgba(7,10,16,0.82); box-shadow: 0 26px 90px rgba(0,0,0,0.48); backdrop-filter: blur(24px); }
.admin-login-visual { position: relative; display: flex; flex-direction: column; justify-content: space-between; padding: 2rem; overflow: hidden; background: linear-gradient(145deg, rgba(127,29,29,0.18), rgba(15,23,42,0.08)), linear-gradient(180deg, rgba(255,255,255,0.07), transparent); }
.admin-login-visual::after { content: ""; position: absolute; inset: auto -18% -28% 8%; height: 62%; transform: rotate(-7deg); border: 1px solid rgba(255,255,255,0.08); border-radius: 42px; background: linear-gradient(135deg, rgba(248,113,113,0.12), rgba(34,211,238,0.08)); }
.admin-login-mark { position: relative; z-index: 1; display: flex; align-items: center; gap: 1rem; color: #f8fafc; }
.admin-login-mark-icon { display: grid; place-items: center; width: 56px; height: 56px; border: 1px solid rgba(248,113,113,0.24); border-radius: 16px; background: rgba(248,113,113,0.12); color: #fca5a5; }
.admin-login-mark-icon .iconify { position: relative; border: 2px solid currentColor; border-radius: 8px; }
.admin-login-mark-icon .iconify::after { content: ""; position: absolute; inset: 5px; border-right: 2px solid currentColor; border-bottom: 2px solid currentColor; transform: rotate(38deg); }
.admin-login-eyebrow { margin: 0 0 0.25rem; color: #67e8f9; font-size: 0.72rem; font-weight: 800; text-transform: uppercase; letter-spacing: 0; }
.admin-login-mark h2 { max-width: 10ch; margin: 0; color: #f8fafc; font-size: 4.1rem; line-height: 0.94; font-weight: 850; letter-spacing: 0; }
.admin-login-command { position: relative; z-index: 1; width: min(100%, 410px); padding: 1rem; border: 1px solid rgba(255,255,255,0.1); border-radius: 18px; background: rgba(3,7,13,0.78); box-shadow: inset 0 1px 0 rgba(255,255,255,0.06), 0 18px 48px rgba(0,0,0,0.3); font-family: 'SF Mono', ui-monospace, monospace; }
.admin-login-command-head { display: flex; gap: 0.45rem; margin-bottom: 1rem; }
.admin-login-command-head span { width: 0.62rem; height: 0.62rem; border-radius: 999px; background: #fb7185; }
.admin-login-command-head span:nth-child(2) { background: #fbbf24; }
.admin-login-command-head span:nth-child(3) { background: #22d3ee; }
.admin-login-command-row { display: flex; gap: 0.65rem; min-width: 0; color: rgba(255,255,255,0.74); font-size: 0.82rem; line-height: 1.9; }
.admin-login-command-row .prompt { flex: 0 0 auto; color: #67e8f9; }
.admin-login-command-row.muted { color: rgba(255,255,255,0.48); }
.admin-login-command-row.success { color: #fca5a5; }
.admin-login-stats { position: relative; z-index: 1; display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.75rem; }
.admin-login-stats div { min-width: 0; padding: 0.85rem; border: 1px solid rgba(255,255,255,0.1); border-radius: 14px; background: rgba(255,255,255,0.06); }
.admin-login-stats span { display: block; margin-bottom: 0.35rem; color: rgba(255,255,255,0.48); font-size: 0.68rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0; }
.admin-login-stats strong { color: #f8fafc; font-size: 1rem; }
.admin-login-panel { display: grid; place-items: center; padding: 2rem; background: rgba(5,8,14,0.72); }
.admin-login-card { width: 100%; max-width: 370px; }
.admin-login-brand { text-align: left; margin-bottom: 1.75rem; }
.admin-login-icon { display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; margin-bottom: 1rem; border: 1px solid rgba(34,211,238,0.18); border-radius: 14px; background: rgba(34,211,238,0.1); color: #67e8f9; box-shadow: 0 14px 34px rgba(34,211,238,0.1); }
.admin-login-icon .iconify { position: relative; border: 2px solid currentColor; border-radius: 6px; }
.admin-login-icon .iconify::before, .admin-login-icon .iconify::after { content: ""; position: absolute; left: 5px; right: 5px; height: 2px; border-radius: 999px; background: currentColor; }
.admin-login-icon .iconify::before { top: 7px; }
.admin-login-icon .iconify::after { bottom: 7px; }
.admin-login-title { margin: 0; color: #f8fafc; font-size: 1.85rem; line-height: 1.12; font-weight: 850; letter-spacing: 0; }
.admin-login-desc { max-width: 28ch; margin: 0.4rem 0 0; color: rgba(255,255,255,0.6); font-size: 0.92rem; line-height: 1.55; }
.admin-login-form { display: flex; flex-direction: column; gap: 1rem; }
.admin-login-field { display: flex; flex-direction: column; gap: 0.45rem; }
.admin-login-label { color: rgba(255,255,255,0.62); font-size: 0.74rem; font-weight: 750; text-transform: uppercase; letter-spacing: 0; }
.admin-login-input-wrap { position: relative; display: flex; align-items: center; }
.admin-login-input-icon { position: absolute; left: 0.9rem; color: rgba(255,255,255,0.38); pointer-events: none; }
.admin-login-input-icon::before { content: ""; display: block; width: 0.65rem; height: 0.65rem; margin: 0.15rem auto 0; border: 2px solid currentColor; border-radius: 999px; }
.admin-login-input-icon::after { content: ""; display: block; width: 0.85rem; height: 0.38rem; margin: 0.05rem auto 0; border: 2px solid currentColor; border-top: 0; border-radius: 0 0 999px 999px; }
.admin-login-input { width: 100%; height: 46px; padding: 0 0.95rem; border: 1px solid rgba(255,255,255,0.12); border-radius: 10px; background: rgba(255,255,255,0.055); color: #f8fafc; font-size: 0.92rem; outline: none; transition: border-color 0.2s ease, background 0.2s ease, box-shadow 0.2s ease; }
.admin-login-input--with-icon { padding-left: 2.55rem; }
.admin-login-input::placeholder { color: rgba(255,255,255,0.3); }
.admin-login-input:focus { border-color: rgba(34,211,238,0.58); background: rgba(34,211,238,0.06); box-shadow: 0 0 0 3px rgba(34,211,238,0.12); }
.admin-login-input:disabled { opacity: 0.5; cursor: not-allowed; }
.admin-login-input--totp { height: 54px; text-align: center; font-size: 1.55rem; font-family: 'SF Mono', ui-monospace, monospace; letter-spacing: 0; }
.admin-login-input--password { padding-right: 3rem; }
.admin-login-eye { position: absolute; right: 0.45rem; display: grid; place-items: center; width: 34px; height: 34px; border: 0; border-radius: 8px; background: transparent; color: rgba(255,255,255,0.48); cursor: pointer; transition: color 0.2s ease, background 0.2s ease; }
.admin-login-eye .iconify { position: relative; border: 2px solid currentColor; border-radius: 999px / 70%; }
.admin-login-eye .iconify::after { content: ""; position: absolute; inset: 5px; border-radius: 999px; background: currentColor; }
.admin-login-eye:hover { color: #f8fafc; background: rgba(255,255,255,0.08); }
.admin-login-error { display: flex; align-items: center; gap: 0.5rem; padding: 0.7rem 0.85rem; border: 1px solid rgba(248,113,113,0.24); border-radius: 10px; background: rgba(248,113,113,0.1); color: #fca5a5; font-size: 0.82rem; }
.admin-login-btn { display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; width: 100%; height: 46px; margin-top: 0.2rem; border: 0; border-radius: 10px; background: linear-gradient(135deg, #ef4444, #0891b2); color: #fff; font-size: 0.9rem; font-weight: 850; cursor: pointer; box-shadow: 0 14px 30px rgba(239,68,68,0.18); transition: filter 0.2s ease, box-shadow 0.2s ease, transform 0.1s ease; }
.admin-login-btn .iconify:not(.admin-login-spin)::before { content: ">"; display: block; color: currentColor; font-weight: 900; line-height: 1; }
.admin-login-btn:hover:not(:disabled) { filter: brightness(1.08); box-shadow: 0 18px 38px rgba(8,145,178,0.24); }
.admin-login-btn:active:not(:disabled) { transform: translateY(1px); }
.admin-login-btn:disabled { opacity: 0.58; cursor: not-allowed; }
.admin-login-spin { animation: admin-login-spin 0.85s linear infinite; }
.admin-login-link { display: block; width: 100%; padding: 0.15rem 0; border: 0; background: none; color: rgba(103,232,249,0.82); font-size: 0.82rem; text-align: center; cursor: pointer; transition: color 0.2s ease; }
.admin-login-link:hover { color: #fca5a5; }
.admin-login-meta { display: flex; align-items: center; justify-content: center; gap: 0.5rem; margin-top: 1.25rem; padding-top: 1.25rem; border-top: 1px solid rgba(255,255,255,0.08); color: rgba(255,255,255,0.36); font-family: 'SF Mono', ui-monospace, monospace; font-size: 0.72rem; letter-spacing: 0; }

@keyframes admin-login-spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 840px) {
  .admin-login-root { align-items: start; padding: 1rem; }
  .admin-login-shell { grid-template-columns: 1fr; width: min(100%, 470px); min-height: auto; border-radius: 20px; }
  .admin-login-visual { min-height: 270px; padding: 1.35rem; }
  .admin-login-mark h2 { max-width: 12ch; font-size: 2.7rem; }
  .admin-login-command { display: none; }
  .admin-login-panel { padding: 1.5rem; }
  .admin-login-card { max-width: none; }
}

@media (max-width: 420px) {
  .admin-login-root { padding: 0; }
  .admin-login-shell { min-height: 100vh; border: 0; border-radius: 0; }
  .admin-login-visual { min-height: 230px; }
  .admin-login-stats { gap: 0.5rem; }
  .admin-login-stats div { padding: 0.7rem; }
  .admin-login-panel { padding: 1.25rem; }
}
</style>
