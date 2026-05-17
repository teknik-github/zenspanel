<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { terminalApi } from '@/api/terminal'

const auth = useAuthStore()
const terminalContainer = ref<HTMLDivElement>()
const connected = ref(false)
const disconnected = ref(false)
let ws: WebSocket | null = null
let term: any = null
let fitAddon: any = null

async function connect() {
  if (typeof window === 'undefined') return
  const { Terminal } = await import('xterm')
  const { FitAddon } = await import('xterm-addon-fit')
  await import('xterm/css/xterm.css')

  term = new Terminal({ cursorBlink: true, fontSize: 13, theme: { background: '#1e293b' } })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(terminalContainer.value!)
  fitAddon.fit()

  const res = await terminalApi.getToken()
  const token = res.data.token

  ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws/terminal?token=${token}`)

  ws.onopen = () => { connected.value = true; disconnected.value = false }
  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data)
      if (msg.type === 'output') term.write(msg.data)
    } catch { term.write(e.data) }
  }
  ws.onclose = () => { connected.value = false; disconnected.value = true }

  term.onData((data: string) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }))
    }
  })

  window.addEventListener('resize', () => fitAddon?.fit())
}

onMounted(() => {
  if (auth.user?.terminal_enabled) connect()
})

onUnmounted(() => {
  ws?.close()
  term?.dispose()
})
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center justify-between mb-3">
      <h1 class="text-lg font-semibold text-gray-800">Terminal</h1>
      <div class="flex items-center gap-2">
        <span class="flex items-center gap-1.5 text-xs"
          :class="connected ? 'text-green-600' : 'text-gray-400'">
          <span class="w-2 h-2 rounded-full" :class="connected ? 'bg-green-500' : 'bg-gray-300'"></span>
          {{ connected ? 'Connected' : 'Disconnected' }}
        </span>
        <button v-if="disconnected" @click="connect"
          class="text-xs border border-gray-200 px-3 py-1.5 rounded-md text-gray-600 hover:bg-gray-50">
          Reconnect
        </button>
      </div>
    </div>

    <div v-if="!auth.user?.terminal_enabled"
      class="flex-1 flex items-center justify-center bg-white border border-gray-200 rounded-lg">
      <div class="text-center text-gray-400">
        <svg class="w-10 h-10 mx-auto mb-2 opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
        </svg>
        <p class="text-sm">Terminal is disabled for your account.</p>
        <p class="text-xs mt-1">Contact your administrator to enable it.</p>
      </div>
    </div>

    <div v-else ref="terminalContainer" class="flex-1 rounded-lg overflow-hidden bg-slate-800 min-h-[400px]"></div>
  </div>
</template>
