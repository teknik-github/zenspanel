<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { terminalApi } from '@/api/terminal'

const auth = useAuthStore()
const terminalContainer = ref<HTMLDivElement>()
const connected = ref(false)
const disconnected = ref(false)
const connecting = ref(false)
let ws: WebSocket | null = null
let term: any = null
let fitAddon: any = null
// Hold the resize handler in a module-scoped ref so we can detach it on
// unmount AND between reconnects — without this, every Reconnect click
// added another listener to window and caused fitAddon to be re-fired N
// times per resize.
let resizeHandler: (() => void) | null = null

async function connect() {
  if (typeof window === 'undefined') return
  if (connecting.value || connected.value) return
  connecting.value = true
  try {
    const { Terminal } = await import('xterm')
    const { FitAddon } = await import('xterm-addon-fit')
    await import('xterm/css/xterm.css')

    // If we're reconnecting, dispose the previous term so xterm doesn't
    // attach a second instance into the same container.
    if (term) {
      term.dispose()
      term = null
    }

    term = new Terminal({ cursorBlink: true, fontSize: 13, theme: { background: '#1e293b' } })
    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(terminalContainer.value!)
    fitAddon.fit()

    const res = await terminalApi.getToken()
    const token = res.data.token

    ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws/terminal?token=${token}`)

    ws.onopen = () => { connected.value = true; disconnected.value = false; connecting.value = false }
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        if (msg.type === 'output') term.write(msg.data)
      } catch { term.write(e.data) }
    }
    ws.onclose = () => { connected.value = false; disconnected.value = true; connecting.value = false }
    ws.onerror = () => { connecting.value = false }

    term.onData((data: string) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }))
      }
    })

    if (resizeHandler) window.removeEventListener('resize', resizeHandler)
    resizeHandler = () => fitAddon?.fit()
    window.addEventListener('resize', resizeHandler)
  } catch (e) {
    connecting.value = false
    disconnected.value = true
  }
}

onMounted(() => {
  if (auth.user?.terminal_enabled) connect()
})

onUnmounted(() => {
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler)
    resizeHandler = null
  }
  ws?.close()
  ws = null
  term?.dispose()
  term = null
})
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center justify-between mb-3 gap-3">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Terminal</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">SSH-style shell access to your account</p>
      </div>
      <div class="flex items-center gap-2 flex-shrink-0">
        <span class="flex items-center gap-1.5 text-xs"
          :class="connecting ? 'text-amber-600' : connected ? 'text-green-600' : 'text-gray-400'">
          <span class="w-2 h-2 rounded-full"
            :class="connecting ? 'bg-amber-500 animate-pulse' : connected ? 'bg-green-500' : 'bg-gray-300'"></span>
          {{ connecting ? 'Connecting…' : connected ? 'Connected' : 'Disconnected' }}
        </span>
        <button v-if="disconnected && !connecting" @click="connect"
          class="text-xs border border-gray-200 px-3 py-1.5 rounded-md text-gray-600 hover:bg-gray-50 transition-colors">
          Reconnect
        </button>
      </div>
    </div>

    <div v-if="!auth.user?.terminal_enabled"
      class="flex-1 bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-16 px-4 text-center">
      <div class="w-12 h-12 rounded-full bg-gray-100 text-gray-400 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">Terminal is disabled</p>
      <p class="text-xs text-gray-400 mt-1">Contact your administrator to enable shell access</p>
    </div>

    <div v-else class="flex-1 relative rounded-lg overflow-hidden bg-slate-800 min-h-[400px]">
      <div ref="terminalContainer" class="absolute inset-0"></div>
      <div v-if="connecting && !connected"
        class="absolute inset-0 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm pointer-events-none">
        <div class="text-center text-slate-200">
          <svg class="w-6 h-6 mx-auto animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          <p class="text-xs mt-2">Connecting to shell…</p>
        </div>
      </div>
    </div>
  </div>
</template>
