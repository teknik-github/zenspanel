<script setup lang="ts">
import type { FitAddon as XTermFitAddon } from '@xterm/addon-fit'
import type { IDisposable, ITerminalOptions, Terminal as XTermTerminal } from '@xterm/xterm'

definePageMeta({ alias: '/admin/terminal' })

type TerminalTokenResponse = {
  token?: string
}

type TerminalOutputMessage = {
  type?: string
  data?: string
}

type ApiError = {
  data?: {
    error?: string
  }
}

type ConnectionState = 'idle' | 'connecting' | 'connected' | 'closed' | 'error'

const auth = useAuth()
const toast = useToast()
const isAdmin = computed(() => auth.user.value?.role === 'admin')

const username = ref('')
const loading = ref(false)
const connectionState = ref<ConnectionState>('idle')
const terminalContainer = ref<HTMLElement | null>(null)
const activeTarget = ref('')
const terminalSize = reactive({ cols: 0, rows: 0 })

let terminal: XTermTerminal | null = null
let fitAddon: XTermFitAddon | null = null
let socket: WebSocket | null = null
let inputDisposable: IDisposable | null = null
let resizeDisposable: IDisposable | null = null
let resizeObserver: ResizeObserver | null = null

const statusColor = computed(() => {
  if (connectionState.value === 'connected') {
    return 'success'
  }

  if (connectionState.value === 'connecting') {
    return 'warning'
  }

  if (connectionState.value === 'error') {
    return 'error'
  }

  return 'neutral'
})

const statusLabel = computed(() => {
  const labels: Record<ConnectionState, string> = {
    idle: 'Ready',
    connecting: 'Connecting',
    connected: 'Connected',
    closed: 'Closed',
    error: 'Error'
  }

  return labels[connectionState.value]
})

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Request failed.'
}

function buildWebSocketUrl(token: string) {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${location.host}/ws/terminal?token=${encodeURIComponent(token)}`
}

function decodeBase64(data: string) {
  const binary = atob(data)
  const bytes = new Uint8Array(binary.length)

  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }

  return bytes
}

function sendMessage(message: Record<string, unknown>) {
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify(message))
  }
}

function sendResize(cols: number, rows: number) {
  terminalSize.cols = cols
  terminalSize.rows = rows
  sendMessage({ type: 'resize', cols, rows })
}

function fitTerminal() {
  if (!fitAddon || !terminal || !terminalContainer.value) {
    return
  }

  try {
    fitAddon.fit()
    sendResize(terminal.cols, terminal.rows)
  } catch {
    // xterm throws if the container has not been measured yet.
  }
}

async function initializeTerminal() {
  if (terminal || !terminalContainer.value) {
    return
  }

  const [{ Terminal }, { FitAddon }] = await Promise.all([
    import('@xterm/xterm'),
    import('@xterm/addon-fit')
  ])

  const options: ITerminalOptions = {
    cursorBlink: true,
    convertEol: true,
    scrollback: 5000,
    fontFamily: '"JetBrains Mono", "SFMono-Regular", Consolas, monospace',
    fontSize: 13,
    lineHeight: 1.15,
    theme: {
      background: '#05080d',
      foreground: '#d6deeb',
      cursor: '#22d3ee',
      selectionBackground: '#233044'
    }
  }

  terminal = markRaw(new Terminal(options))
  fitAddon = markRaw(new FitAddon())
  terminal.loadAddon(fitAddon)
  terminal.open(terminalContainer.value)

  inputDisposable = terminal.onData((data) => {
    sendMessage({ type: 'input', data })
  })

  resizeDisposable = terminal.onResize(({ cols, rows }) => {
    sendResize(cols, rows)
  })

  terminal.writeln('Ready. Click Connect to open a terminal session.')
  await nextTick()
  fitTerminal()
}

async function mintToken() {
  const endpoint = isAdmin.value ? '/api/v1/admin/terminal/token' : '/api/v1/terminal/token'
  const body: Record<string, string> = {}

  if (isAdmin.value && username.value.trim()) {
    body.username = username.value.trim()
  }

  const response = await $fetch<TerminalTokenResponse>(endpoint, {
    method: 'POST',
    body
  })

  if (!response.token) {
    throw new Error('Terminal token was not returned.')
  }

  return response.token
}

async function connectTerminal() {
  if (loading.value || connectionState.value === 'connected') {
    return
  }

  loading.value = true
  connectionState.value = 'connecting'

  try {
    await initializeTerminal()
    const token = await mintToken()
    const wsUrl = buildWebSocketUrl(token)
    const ws = new WebSocket(wsUrl)

    disconnectTerminal(false)
    socket = ws
    activeTarget.value = isAdmin.value && username.value.trim()
      ? username.value.trim()
      : isAdmin.value ? 'zenspanel' : auth.user.value?.username || 'account'

    terminal?.reset()
    terminal?.writeln('Connecting...')

    ws.addEventListener('open', () => {
      if (socket !== ws) {
        return
      }

      connectionState.value = 'connected'
      terminal?.clear()
      fitTerminal()
      terminal?.focus()
      toast.add({ title: 'Terminal connected', color: 'success' })
    })

    ws.addEventListener('message', (event: MessageEvent<string>) => {
      if (socket !== ws) {
        return
      }

      try {
        const message = JSON.parse(event.data) as TerminalOutputMessage

        if (message.type === 'output' && message.data) {
          terminal?.write(decodeBase64(message.data))
        }
      } catch {
        terminal?.writeln('\r\n[invalid terminal frame]')
      }
    })

    ws.addEventListener('close', () => {
      if (socket !== ws) {
        return
      }

      socket = null
      connectionState.value = connectionState.value === 'error' ? 'error' : 'closed'
      terminal?.writeln('\r\n[session closed]')
    })

    ws.addEventListener('error', () => {
      if (socket !== ws) {
        return
      }

      connectionState.value = 'error'
      terminal?.writeln('\r\n[connection error]')
      toast.add({
        title: 'Terminal connection failed',
        color: 'error'
      })
    })
  } catch (error: unknown) {
    connectionState.value = 'error'
    terminal?.writeln(`\r\n[error] ${getErrorMessage(error)}`)
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

function disconnectTerminal(writeMessage = true) {
  if (socket) {
    socket.close()
    socket = null
  }

  if (writeMessage && connectionState.value === 'connected') {
    terminal?.writeln('\r\n[disconnect requested]')
  }

  if (writeMessage) {
    connectionState.value = 'closed'
  }
}

onMounted(async () => {
  await initializeTerminal()

  if (terminalContainer.value) {
    resizeObserver = new ResizeObserver(() => fitTerminal())
    resizeObserver.observe(terminalContainer.value)
  }
})

onBeforeUnmount(() => {
  disconnectTerminal(false)
  resizeObserver?.disconnect()
  inputDisposable?.dispose()
  resizeDisposable?.dispose()
  terminal?.dispose()
})
</script>

<template>
  <UDashboardPanel id="terminal">
    <template #header>
      <UDashboardNavbar :title="isAdmin ? 'Terminal (Admin)' : 'Terminal'">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UBadge :color="statusColor" variant="subtle">
            {{ statusLabel }}
          </UBadge>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex min-h-[620px] flex-col gap-4 lg:h-[calc(100vh-8rem)]">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div class="flex flex-wrap items-end gap-3">
            <UFormField
              v-if="isAdmin"
              label="Run as user"
              class="w-64"
            >
              <UInput
                v-model="username"
                icon="i-lucide-user"
                :disabled="loading || connectionState === 'connected'"
                placeholder="zenspanel"
              />
            </UFormField>

            <UBadge color="neutral" variant="subtle">
              {{ activeTarget || (isAdmin ? 'zenspanel' : auth.user.value?.username || 'account') }}
            </UBadge>

            <UBadge
              v-if="terminalSize.cols && terminalSize.rows"
              color="neutral"
              variant="subtle"
            >
              {{ terminalSize.cols }}x{{ terminalSize.rows }}
            </UBadge>
          </div>

          <div class="flex gap-2">
            <UButton
              v-if="connectionState === 'connected'"
              label="Disconnect"
              icon="i-lucide-unplug"
              color="neutral"
              variant="subtle"
              @click="disconnectTerminal()"
            />
            <UButton
              v-else
              label="Connect"
              icon="i-lucide-terminal"
              :loading="loading"
              @click="connectTerminal"
            />
          </div>
        </div>

        <div class="terminal-frame flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-default bg-[#05080d]">
          <div class="flex items-center justify-between border-b border-white/10 px-4 py-2">
            <div class="flex items-center gap-2">
              <span class="size-2.5 rounded-full bg-error" />
              <span class="size-2.5 rounded-full bg-warning" />
              <span class="size-2.5 rounded-full bg-success" />
            </div>

            <span class="font-mono text-xs text-white/45">
              /ws/terminal
            </span>
          </div>

          <div ref="terminalContainer" class="terminal-surface min-h-0 flex-1 p-3" />
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>

<style scoped>
.terminal-frame {
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 0.05), 0 18px 40px rgb(0 0 0 / 0.18);
}

.terminal-surface :deep(.xterm) {
  height: 100%;
}

.terminal-surface :deep(.xterm-viewport) {
  overflow-y: auto;
}
</style>
