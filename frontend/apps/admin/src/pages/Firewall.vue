<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { firewallApi } from '@/api/firewall'

const blockedIPs = ref<any[]>([])
const jails = ref<any[]>([])
const loadingIPs = ref(false)
const loadingJails = ref(false)
const blockIP = ref('')
const blockReason = ref('')
const blocking = ref(false)
const blockError = ref('')
const confirmUnblock = ref<string | null>(null)
const jailSaving = ref<string | null>(null)

onMounted(async () => {
  await Promise.all([fetchBlocked(), fetchJails()])
})

async function fetchBlocked() {
  loadingIPs.value = true
  try {
    const res = await firewallApi.listBlocked()
    blockedIPs.value = res.data.ips || []
  } finally {
    loadingIPs.value = false
  }
}

async function fetchJails() {
  loadingJails.value = true
  try {
    const res = await firewallApi.listJails()
    jails.value = res.data.jails || []
  } finally {
    loadingJails.value = false
  }
}

async function blockIPSubmit() {
  blockError.value = ''
  if (!blockIP.value.trim()) return
  blocking.value = true
  try {
    await firewallApi.block(blockIP.value.trim(), blockReason.value.trim())
    blockIP.value = ''
    blockReason.value = ''
    await fetchBlocked()
  } catch (e: any) {
    blockError.value = e.response?.data?.error || 'Failed to block IP'
  } finally {
    blocking.value = false
  }
}

async function unblock(ip: string) {
  await firewallApi.unblock(ip)
  confirmUnblock.value = null
  await fetchBlocked()
}

async function toggleJail(jail: any) {
  jailSaving.value = jail.name
  try {
    await firewallApi.setJail(jail.name, !jail.enabled)
    jail.enabled = !jail.enabled
  } finally {
    jailSaving.value = null
  }
}

// Human-readable descriptions for known fail2ban jails
const jailDescriptions: Record<string, string> = {
  'sshd':              'SSH brute-force',
  'vsftpd':            'FTP brute-force',
  'zenspanel-login':   'Panel login brute-force',
  'nginx-http-auth':   'HTTP auth brute-force',
  'nginx-botsearch':   'Bot/scanner detection',
  'recidive':          'Repeat offenders (multi-jail)',
  'postfix':           'Mail spam/abuse',
  'dovecot':           'IMAP/POP3 brute-force',
}

function jailDesc(name: string): string {
  return jailDescriptions[name] || '—'
}
const reasonLabels: Record<string, string> = {
  'fail2ban: sshd':           'SSH brute-force',
  'fail2ban: vsftpd':         'FTP brute-force',
  'fail2ban: zenspanel-login':'Panel login brute-force',
  'fail2ban: nginx-http-auth':'HTTP auth brute-force',
  'fail2ban: nginx-botsearch': 'Bot/scanner detected',
  'fail2ban: recidive':       'Repeat offender (recidive)',
  'fail2ban: postfix':        'Mail spam/abuse',
  'fail2ban: dovecot':        'IMAP/POP3 brute-force',
}

function formatReason(reason: string, source: string): string {
  if (!reason) return source === 'fail2ban' ? 'Auto-banned by fail2ban' : '—'
  return reasonLabels[reason] || reason
}

function reasonIcon(reason: string, source: string): string {
  if (source === 'fail2ban') {
    if (reason?.includes('ssh')) return 'terminal'
    if (reason?.includes('ftp') || reason?.includes('vsftpd')) return 'hard-drive'
    if (reason?.includes('login') || reason?.includes('zenspanel')) return 'lock'
    if (reason?.includes('bot') || reason?.includes('scan')) return 'search'
    if (reason?.includes('recidive')) return 'repeat'
    return 'shield'
  }
  return 'user'
}
</script>

<template>
  <div class="space-y-6">
    <h1 class="text-lg font-semibold text-gray-800">Firewall</h1>

    <!-- Block IP form -->
    <div class="bg-white border border-gray-200 rounded-lg p-4">
      <h2 class="text-sm font-semibold text-gray-800 mb-3">Block IP Address</h2>
      <div class="flex flex-wrap gap-2">
        <input v-model="blockIP" type="text" placeholder="1.2.3.4 or 1.2.3.0/24"
          class="border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500 w-48" />
        <input v-model="blockReason" type="text" placeholder="Reason (optional)"
          class="border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 flex-1 min-w-[160px]" />
        <button @click="blockIPSubmit" :disabled="blocking || !blockIP.trim()"
          class="bg-red-600 text-white text-sm px-4 py-2 rounded-md hover:bg-red-700 disabled:opacity-50">
          {{ blocking ? 'Blocking...' : 'Block' }}
        </button>
      </div>
      <p v-if="blockError" class="text-xs text-red-600 mt-2">{{ blockError }}</p>
    </div>

    <!-- Blocked IPs table -->
    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
        <h2 class="text-sm font-semibold text-gray-800">Blocked IPs</h2>
        <button @click="fetchBlocked" class="text-xs text-gray-400 hover:text-gray-600">Refresh</button>
      </div>

      <div v-if="loadingIPs" class="p-4 space-y-2">
        <div v-for="i in 3" :key="i" class="h-8 bg-gray-50 rounded animate-pulse"></div>
      </div>

      <div v-else-if="!blockedIPs.length" class="flex flex-col items-center justify-center py-10 text-center">
        <svg class="w-8 h-8 text-gray-300 mb-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
        </svg>
        <p class="text-sm text-gray-500">No blocked IPs</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full text-xs min-w-[500px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">IP Address</th>
              <th class="text-left px-4 py-3 font-medium">Reason</th>
              <th class="text-left px-4 py-3 font-medium">Source</th>
              <th class="text-left px-4 py-3 font-medium">Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in blockedIPs" :key="entry.ip"
              class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-mono text-gray-800">{{ entry.ip }}</td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  <!-- SSH -->
                  <svg v-if="reasonIcon(entry.reason, entry.source) === 'terminal'" class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
                  </svg>
                  <!-- FTP -->
                  <svg v-else-if="reasonIcon(entry.reason, entry.source) === 'hard-drive'" class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="22" y1="12" x2="2" y2="12"/><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>
                  </svg>
                  <!-- Login -->
                  <svg v-else-if="reasonIcon(entry.reason, entry.source) === 'lock'" class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                  </svg>
                  <!-- Repeat offender -->
                  <svg v-else-if="reasonIcon(entry.reason, entry.source) === 'repeat'" class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/>
                  </svg>
                  <!-- Default shield -->
                  <svg v-else class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                  </svg>
                  <span class="text-gray-600">{{ formatReason(entry.reason, entry.source) }}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="entry.source === 'fail2ban' ? 'bg-orange-100 text-orange-700' : 'bg-red-100 text-red-700'">
                  {{ entry.source === 'fail2ban' ? 'fail2ban' : 'manual' }}
                </span>
              </td>
              <td class="px-4 py-3">
                <button @click="confirmUnblock = entry.ip"
                  class="text-xs text-indigo-600 border border-indigo-200 px-2 py-1 rounded hover:bg-indigo-50">
                  Unblock
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- fail2ban jails -->
    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-gray-800">fail2ban Jails</h2>
          <p class="text-xs text-gray-400 mt-0.5">Auto-ban IPs that trigger repeated failures.</p>
        </div>
        <button @click="fetchJails" class="text-xs text-gray-400 hover:text-gray-600">Refresh</button>
      </div>

      <div v-if="loadingJails" class="p-4 space-y-2">
        <div v-for="i in 3" :key="i" class="h-8 bg-gray-50 rounded animate-pulse"></div>
      </div>

      <div v-else-if="!jails.length" class="px-4 py-8 text-center text-sm text-gray-400">
        fail2ban not running or no jails configured.
      </div>

      <table v-else class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Jail</th>
            <th class="text-left px-4 py-3 font-medium">Protects Against</th>
            <th class="text-left px-4 py-3 font-medium">Status</th>
            <th class="text-left px-4 py-3 font-medium">Currently Banned</th>
            <th class="text-left px-4 py-3 font-medium">Total Bans</th>
            <th class="text-left px-4 py-3 font-medium">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="jail in jails" :key="jail.name"
            class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-mono text-gray-800">{{ jail.name }}</td>
            <td class="px-4 py-3 text-gray-500">{{ jailDesc(jail.name) }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                :class="jail.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
                {{ jail.enabled ? 'Active' : 'Disabled' }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-600">{{ jail.currently_banned ?? 0 }}</td>
            <td class="px-4 py-3 text-gray-600">{{ jail.ban_count ?? 0 }}</td>
            <td class="px-4 py-3">
              <button @click="toggleJail(jail)" :disabled="jailSaving === jail.name"
                class="text-xs px-3 py-1 rounded border transition-colors disabled:opacity-50"
                :class="jail.enabled
                  ? 'text-red-600 border-red-200 hover:bg-red-50'
                  : 'text-green-600 border-green-200 hover:bg-green-50'">
                {{ jailSaving === jail.name ? '...' : jail.enabled ? 'Disable' : 'Enable' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Confirm unblock dialog -->
    <div v-if="confirmUnblock" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Unblock {{ confirmUnblock }}?</h2>
        <p class="text-sm text-gray-500 mb-4">This will remove the IP from the block list immediately.</p>
        <div class="flex gap-2">
          <button @click="confirmUnblock = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="unblock(confirmUnblock!)"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700">Unblock</button>
        </div>
      </div>
    </div>
  </div>
</template>
