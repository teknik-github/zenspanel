<script setup lang="ts">
definePageMeta({
  layout: false
})

const auth = useAuth()
const router = useRouter()

function dashboardPath() {
  return auth.user.value?.role === 'admin' ? '/admin/dashboard' : '/dashboard'
}

const username = ref('')
const password = ref('')
const mounted = ref(false)
const showPassword = ref(false)

onMounted(() => {
  requestAnimationFrame(() => {
    mounted.value = true
  })
})

if (auth.isAuthenticated.value) {
  await navigateTo(dashboardPath())
}

async function handleLogin() {
  try {
    await auth.login(username.value, password.value)
    if (!auth.requires2FA.value) {
      await navigateTo(dashboardPath())
    }
  } catch {
    // Error is set in auth.error
  }
}

// 2FA flow
const totpCode = ref('')
const recoveryCode = ref('')
const showRecovery = ref(false)

async function handleVerify2FA() {
  try {
    await auth.verify2FA(totpCode.value)
    await navigateTo(dashboardPath())
  } catch {
    // Error is set in auth.error
  }
}
async function handleRecover2FA() {
  try {
    await auth.recover2FA(recoveryCode.value)
    await navigateTo(dashboardPath())
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
    router.push(dashboardPath())
  }
})
</script>

<template>
  <div class="user-login-root">
    <div class="user-login-grid" />
    <div class="user-login-shell" :class="{ 'user-login-shell--visible': mounted }">
      <section class="user-login-showcase" aria-hidden="true">
        <div class="user-login-brandmark">
          <div class="user-login-brandmark-icon">
            <UIcon name="i-lucide-panel-top" class="size-7" />
          </div>
          <div>
            <p class="user-login-kicker">
              ZensPanel
            </p>
            <h2>Hosting control center</h2>
          </div>
        </div>

        <div class="user-login-console">
          <div class="user-login-console-top">
            <span />
            <span />
            <span />
          </div>
          <div class="user-login-console-lines">
            <span class="line line--wide" />
            <span class="line line--short" />
            <span class="line line--medium" />
            <span class="line line--wide muted" />
          </div>
        </div>

        <div class="user-login-metrics">
          <div>
            <span>CPU</span>
            <strong>18%</strong>
          </div>
          <div>
            <span>RAM</span>
            <strong>2.4G</strong>
          </div>
          <div>
            <span>SSL</span>
            <strong>OK</strong>
          </div>
        </div>
      </section>

      <section class="user-login-panel">
        <!-- 2FA step -->
        <div v-if="auth.requires2FA.value" class="user-login-card">
          <div class="user-login-inner">
            <div class="user-login-brand">
              <div class="user-login-icon">
                <UIcon name="i-lucide-shield-check" class="size-6" />
              </div>
              <h1 class="user-login-title">
                Verify Identity
              </h1>
              <p class="user-login-desc">
                Enter the code from your authenticator app
              </p>
            </div>
            <form v-if="!showRecovery" class="user-login-form" @submit.prevent="handleVerify2FA">
              <div class="user-login-field">
                <label class="user-login-label">Auth Code</label>
                <input
                  v-model="totpCode"
                  type="text"
                  inputmode="numeric"
                  autocomplete="one-time-code"
                  placeholder="000000"
                  maxlength="6"
                  required
                  :disabled="auth.loading.value"
                  class="user-login-input user-login-input--totp"
                >
              </div>
              <div v-if="auth.error.value" class="user-login-error">
                {{ auth.error.value }}
              </div>
              <button type="submit" :disabled="auth.loading.value" class="user-login-btn">
                <UIcon v-if="auth.loading.value" name="i-lucide-loader-circle" class="size-4 user-login-spin" />
                <UIcon v-else name="i-lucide-check" class="size-4" />
                <span>{{ auth.loading.value ? 'Verifying...' : 'Verify' }}</span>
              </button>
              <button type="button" class="user-login-link" @click="toggleRecovery">
                Use recovery code
              </button>
            </form>
            <form v-else class="user-login-form" @submit.prevent="handleRecover2FA">
              <div class="user-login-field">
                <label class="user-login-label">Recovery Code</label>
                <input
                  v-model="recoveryCode"
                  type="text"
                  placeholder="XXXX-XXXX-XXXX-XXXX"
                  required
                  :disabled="auth.loading.value"
                  class="user-login-input"
                >
              </div>
              <div v-if="auth.error.value" class="user-login-error">
                {{ auth.error.value }}
              </div>
              <button type="submit" :disabled="auth.loading.value" class="user-login-btn">
                <UIcon v-if="auth.loading.value" name="i-lucide-loader-circle" class="size-4 user-login-spin" />
                <UIcon v-else name="i-lucide-key-round" class="size-4" />
                <span>{{ auth.loading.value ? 'Recovering...' : 'Recover Access' }}</span>
              </button>
              <button type="button" class="user-login-link" @click="toggleRecovery">
                Back to verification code
              </button>
            </form>
          </div>
        </div>

        <!-- Login form -->
        <div v-else class="user-login-card">
          <div class="user-login-inner">
            <div class="user-login-brand">
              <div class="user-login-icon">
                <UIcon name="i-lucide-globe-2" class="size-6" />
              </div>
              <h1 class="user-login-title">
                ZensPanel
              </h1>
              <p class="user-login-desc">
                Sign in to manage your hosting services
              </p>
            </div>
            <form class="user-login-form" @submit.prevent="handleLogin">
              <div class="user-login-field">
                <label class="user-login-label" for="lu">Username</label>
                <div class="user-login-input-wrap">
                  <UIcon name="i-lucide-user" class="size-4 user-login-input-icon" />
                  <input
                    id="lu"
                    v-model="username"
                    type="text"
                    autocomplete="username"
                    placeholder="username"
                    required
                    :disabled="auth.loading.value"
                    class="user-login-input user-login-input--with-icon"
                  >
                </div>
              </div>
              <div class="user-login-field">
                <label class="user-login-label" for="lp">Password</label>
                <div class="user-login-input-wrap">
                  <UIcon name="i-lucide-lock-keyhole" class="size-4 user-login-input-icon" />
                  <input
                    id="lp"
                    v-model="password"
                    :type="showPassword ? 'text' : 'password'"
                    autocomplete="current-password"
                    placeholder="password"
                    required
                    :disabled="auth.loading.value"
                    class="user-login-input user-login-input--with-icon user-login-input--password"
                  >
                  <button
                    type="button"
                    class="user-login-eye"
                    :aria-label="showPassword ? 'Hide password' : 'Show password'"
                    @click="showPassword = !showPassword"
                  >
                    <UIcon :name="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="size-4" />
                  </button>
                </div>
              </div>
              <div v-if="auth.error.value" class="user-login-error">
                <UIcon name="i-lucide-circle-alert" class="size-4 shrink-0" /> {{ auth.error.value }}
              </div>
              <button type="submit" :disabled="auth.loading.value" class="user-login-btn">
                <UIcon v-if="auth.loading.value" name="i-lucide-loader-circle" class="size-4 user-login-spin" />
                <UIcon v-else name="i-lucide-log-in" class="size-4" />
                <span>{{ auth.loading.value ? 'Signing in...' : 'Sign In' }}</span>
              </button>
            </form>
            <div class="user-login-meta">
              <span>ZensPanel v2.0.0</span><span class="user-login-meta-sep">/</span><span>Client Panel</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.user-login-root { position: relative; min-height: 100vh; display: grid; place-items: center; padding: 2rem; overflow: hidden; background: linear-gradient(135deg, #08111f 0%, #10131a 42%, #18100d 100%); font-family: 'Public Sans', ui-sans-serif, system-ui, sans-serif; }
.user-login-root::before { content: ""; position: absolute; inset: 0; background: linear-gradient(120deg, rgba(20,184,166,0.16), transparent 34%), linear-gradient(300deg, rgba(245,158,11,0.12), transparent 42%); }
.user-login-grid { position: absolute; inset: 0; background-image: linear-gradient(rgba(255,255,255,0.045) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.045) 1px, transparent 1px); background-size: 42px 42px; mask-image: linear-gradient(to bottom, transparent, black 18%, black 78%, transparent); }
.user-login-shell { position: relative; z-index: 1; display: grid; grid-template-columns: minmax(0, 1fr) 420px; width: min(100%, 980px); min-height: 600px; overflow: hidden; border: 1px solid rgba(255,255,255,0.12); border-radius: 24px; background: rgba(11,15,22,0.78); box-shadow: 0 24px 80px rgba(0,0,0,0.45); backdrop-filter: blur(24px); opacity: 1; transform: translateY(0) scale(1); transition: box-shadow 0.3s ease, border-color 0.3s ease; }
.user-login-shell--visible { border-color: rgba(255,255,255,0.14); }
.user-login-showcase { position: relative; display: flex; flex-direction: column; justify-content: space-between; padding: 2rem; overflow: hidden; background: linear-gradient(145deg, rgba(15,118,110,0.2), rgba(15,23,42,0.05)), linear-gradient(180deg, rgba(255,255,255,0.08), transparent); }
.user-login-showcase::after { content: ""; position: absolute; inset: auto -16% -28% 10%; height: 62%; transform: rotate(-8deg); border: 1px solid rgba(255,255,255,0.08); border-radius: 42px; background: linear-gradient(135deg, rgba(20,184,166,0.12), rgba(251,191,36,0.08)); }
.user-login-brandmark { position: relative; z-index: 1; display: flex; align-items: center; gap: 1rem; color: #f8fafc; }
.user-login-brandmark-icon { display: grid; place-items: center; width: 54px; height: 54px; border-radius: 16px; background: rgba(20,184,166,0.16); color: #5eead4; border: 1px solid rgba(94,234,212,0.22); }
.user-login-kicker { margin: 0 0 0.25rem; color: #fbbf24; font-size: 0.72rem; font-weight: 800; text-transform: uppercase; letter-spacing: 0; }
.user-login-brandmark h2 { max-width: 11ch; margin: 0; color: #f8fafc; font-size: 4.25rem; line-height: 0.92; font-weight: 800; letter-spacing: 0; }
.user-login-console { position: relative; z-index: 1; width: min(100%, 380px); padding: 1rem; border: 1px solid rgba(255,255,255,0.1); border-radius: 18px; background: rgba(4,8,13,0.72); box-shadow: inset 0 1px 0 rgba(255,255,255,0.06), 0 18px 48px rgba(0,0,0,0.28); }
.user-login-console-top { display: flex; gap: 0.45rem; margin-bottom: 1rem; }
.user-login-console-top span { width: 0.62rem; height: 0.62rem; border-radius: 999px; background: #fb7185; }
.user-login-console-top span:nth-child(2) { background: #fbbf24; }
.user-login-console-top span:nth-child(3) { background: #34d399; }
.user-login-console-lines { display: grid; gap: 0.72rem; }
.line { display: block; height: 0.58rem; border-radius: 999px; background: linear-gradient(90deg, #5eead4, rgba(94,234,212,0.16)); }
.line--wide { width: 92%; }
.line--medium { width: 68%; background: linear-gradient(90deg, #fbbf24, rgba(251,191,36,0.14)); }
.line--short { width: 45%; background: linear-gradient(90deg, #93c5fd, rgba(147,197,253,0.14)); }
.line.muted { opacity: 0.45; }
.user-login-metrics { position: relative; z-index: 1; display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.75rem; }
.user-login-metrics div { min-width: 0; padding: 0.85rem; border: 1px solid rgba(255,255,255,0.1); border-radius: 14px; background: rgba(255,255,255,0.06); }
.user-login-metrics span { display: block; margin-bottom: 0.35rem; color: rgba(255,255,255,0.48); font-size: 0.68rem; font-weight: 700; letter-spacing: 0; }
.user-login-metrics strong { color: #f8fafc; font-size: 1rem; }
.user-login-panel { display: grid; place-items: center; padding: 2rem; background: rgba(8,11,17,0.7); }
.user-login-card { width: 100%; max-width: 360px; }
.user-login-inner { width: 100%; }
.user-login-brand { text-align: left; margin-bottom: 1.75rem; }
.user-login-icon { display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; border-radius: 14px; background: rgba(20,184,166,0.12); color: #5eead4; margin-bottom: 1rem; border: 1px solid rgba(94,234,212,0.16); box-shadow: 0 14px 34px rgba(20,184,166,0.12); }
.user-login-title { margin: 0; font-size: 1.8rem; font-weight: 800; color: #f8fafc; letter-spacing: 0; }
.user-login-desc { max-width: 28ch; font-size: 0.9rem; line-height: 1.55; color: rgba(255,255,255,0.58); margin: 0.35rem 0 0; }
.user-login-form { display: flex; flex-direction: column; gap: 1rem; }
.user-login-field { display: flex; flex-direction: column; gap: 0.45rem; }
.user-login-label { font-size: 0.74rem; font-weight: 700; color: rgba(255,255,255,0.6); letter-spacing: 0; text-transform: uppercase; }
.user-login-input-wrap { position: relative; display: flex; align-items: center; }
.user-login-input-icon { position: absolute; left: 0.9rem; color: rgba(255,255,255,0.36); pointer-events: none; }
.user-login-input { width: 100%; height: 46px; padding: 0 0.95rem; background: rgba(255,255,255,0.055); border: 1px solid rgba(255,255,255,0.12); border-radius: 10px; color: #f8fafc; font-size: 0.92rem; outline: none; transition: border-color 0.2s ease, background 0.2s ease, box-shadow 0.2s ease; }
.user-login-input--with-icon { padding-left: 2.55rem; }
.user-login-input::placeholder { color: rgba(255,255,255,0.28); }
.user-login-input:focus { background: rgba(20,184,166,0.06); border-color: rgba(94,234,212,0.55); box-shadow: 0 0 0 3px rgba(20,184,166,0.12); }
.user-login-input:disabled { opacity: 0.5; cursor: not-allowed; }
.user-login-input--totp { text-align: center; font-size: 1.55rem; letter-spacing: 0; font-family: 'SF Mono', ui-monospace, monospace; height: 54px; }
.user-login-input--password { padding-right: 3rem; }
.user-login-eye { position: absolute; right: 0.45rem; display: grid; place-items: center; width: 34px; height: 34px; border: 0; border-radius: 8px; background: transparent; color: rgba(255,255,255,0.46); cursor: pointer; transition: color 0.2s ease, background 0.2s ease; }
.user-login-eye:hover { color: #f8fafc; background: rgba(255,255,255,0.08); }
.user-login-error { display: flex; align-items: center; gap: 0.5rem; padding: 0.7rem 0.85rem; background: rgba(239,68,68,0.1); border: 1px solid rgba(248,113,113,0.22); border-radius: 10px; color: #fca5a5; font-size: 0.82rem; }
.user-login-btn { display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; width: 100%; height: 46px; margin-top: 0.2rem; background: linear-gradient(135deg, #14b8a6, #0ea5e9); color: white; border: none; border-radius: 10px; font-size: 0.9rem; font-weight: 800; cursor: pointer; box-shadow: 0 14px 30px rgba(14,165,233,0.2); transition: filter 0.2s ease, box-shadow 0.2s ease, transform 0.1s ease; }
.user-login-btn:hover:not(:disabled) { filter: brightness(1.08); box-shadow: 0 18px 38px rgba(20,184,166,0.26); }
.user-login-btn:active:not(:disabled) { transform: translateY(1px); }
.user-login-btn:disabled { opacity: 0.58; cursor: not-allowed; }
.user-login-spin { animation: user-login-spin 0.85s linear infinite; }
.user-login-link { display: block; width: 100%; text-align: center; background: none; border: none; padding: 0.15rem 0; color: rgba(94,234,212,0.78); font-size: 0.82rem; cursor: pointer; transition: color 0.2s ease; }
.user-login-link:hover { color: #fbbf24; }
.user-login-meta { display: flex; align-items: center; justify-content: center; gap: 0.5rem; margin-top: 1.25rem; padding-top: 1.25rem; border-top: 1px solid rgba(255,255,255,0.08); font-size: 0.72rem; color: rgba(255,255,255,0.36); letter-spacing: 0; font-family: 'SF Mono', ui-monospace, monospace; }
.user-login-meta-sep { color: rgba(255,255,255,0.18); }
@keyframes user-login-spin { to { transform: rotate(360deg); } }
@media (max-width: 820px) {
  .user-login-root { padding: 1rem; align-items: start; }
  .user-login-shell { grid-template-columns: 1fr; width: min(100%, 460px); min-height: auto; border-radius: 20px; }
  .user-login-showcase { min-height: 260px; padding: 1.35rem; }
  .user-login-brandmark h2 { max-width: 12ch; font-size: 2.5rem; }
  .user-login-console { display: none; }
  .user-login-panel { padding: 1.5rem; }
  .user-login-card { max-width: none; }
}
@media (max-width: 420px) {
  .user-login-root { padding: 0; }
  .user-login-shell { min-height: 100vh; border: 0; border-radius: 0; }
  .user-login-showcase { min-height: 220px; }
  .user-login-metrics { gap: 0.5rem; }
  .user-login-metrics div { padding: 0.7rem; }
  .user-login-panel { padding: 1.25rem; }
}
</style>
