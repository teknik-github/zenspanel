<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { terminalApi } from '@/api/terminal'
import { usersApi } from '@/api/users'

const terminalContainer = ref<HTMLDivElement>()
const connected = ref(false)
const disconnected = ref(false)
const connecting = ref(false)
const targetUsername = ref('')
const users = ref<any[]>([])
let ws: WebSocket | null = null
let term: any = null
let fitAddon: any = null
let resizeHandler: (() => void) | null = null

onMounted(async () => {
  try {
    const res = await usersApi.list({ limit: 100 })
    users.value = (res.data.data || []).filter((u: any) => u.role === 'user')
  } catch { /* ignore */ }
  connect()
})

onUnmounted(() => {
  if (resizeHandler) window.removeEventListener('resize', resizeHandler)
  ws?.close()
  ws = null
  term?.dispose()
  term = null
})

async function connect() {
  if (connecting.value || connected.value) return
  connecting.value = true
  try {
    const { Terminal } = await import('xterm')
    const { FitAddon } = await import('xterm-addon-fit')
    await import('xterm/css/xterm.css')

    if (term) { term.dispose(); term = null }

    term = new Terminal({ cursorBlink: true, fontSize: 13, theme: { background: '#1e293b' } })
    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(terminalContainer.value!)
    fitAddon.fit()

    const res = await terminalApi.adminToken(targetUsername.value || undefined)
    const token = res.data.token

    ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws/terminal?token=${token}`)
    ws.onopen = () => { connected.value = true; disconnected.value = false; connecting.value = false }
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        if (msg.type === 'output') term.write(atob(msg.data))
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
  } catch {
    connecting.value = false
    disconnected.value = true
  }
}

function reconnect() {
  ws?.close()
  ws = null
  connected.value = false
  disconnected.value = false
  connect()
}
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center justify-between mb-3 gap-3 flex-wrap">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Terminal</h1>
        <p class="text-xs text-gray-400 mt-0.5">Shell access to the server or a specific user account</p>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <select v-model="targetUsername" @change="reconnect"
          class="border border-gray-200 rounded-md px-2 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
          <option value="">zenspanel (system)</option>
          <option v-for="u in users" :key="u.id" :value="u.username">{{ u.username }}</option>
        </select>
        <span class="flex items-center gap-1.5 text-xs"
          :class="connecting ? 'text-amber-600' : connected ? 'text-green-600' : 'text-gray-400'">
          <span class="w-2 h-2 rounded-full"
            :class="connecting ? 'bg-amber-500 animate-pulse' : connected ? 'bg-green-500' : 'bg-gray-300'"></span>
          {{ connecting ? 'Connecting…' : connected ? 'Connected' : 'Disconnected' }}
        </span>
        <button v-if="disconnected && !connecting" @click="reconnect"
          class="text-xs border border-gray-200 px-3 py-1.5 rounded-md text-gray-600 hover:bg-gray-50">
          Reconnect
        </button>
      </div>
    </div>

    <div class="flex-1 relative rounded-lg overflow-hidden bg-slate-800 min-h-[400px]">
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
