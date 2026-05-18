<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, watch } from 'vue'
import { filesApi, type FileEntry } from '@/api/files'

const cwd = ref('')
const entries = ref<FileEntry[]>([])
const selected = ref<FileEntry | null>(null)
const editorContent = ref('')
const dirty = ref(false)
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const showNewFile = ref(false)
const showNewDir = ref(false)
const newName = ref('')
const renameTarget = ref<FileEntry | null>(null)
const renameNewName = ref('')
const confirmDelete = ref<FileEntry | null>(null)

const monaco = shallowRef<any>(null)
const editorContainer = ref<HTMLElement | null>(null)
let editorInstance: any = null

const breadcrumbs = computed(() => {
  const parts = cwd.value.split('/').filter(Boolean)
  const out: { label: string; path: string }[] = [{ label: 'home', path: '' }]
  let acc = ''
  for (const p of parts) {
    acc = acc ? acc + '/' + p : p
    out.push({ label: p, path: acc })
  }
  return out
})

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    const res = await filesApi.list(cwd.value)
    entries.value = (res.data.entries || []).sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
      return a.name.localeCompare(b.name)
    })
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load directory'
  } finally {
    loading.value = false
  }
}

function joinPath(...parts: string[]) {
  return parts.filter(Boolean).join('/')
}

async function openEntry(e: FileEntry) {
  if (e.is_dir) {
    cwd.value = joinPath(cwd.value, e.name)
    selected.value = null
    closeEditor()
    return
  }
  await loadFileIntoEditor(e)
}

async function loadFileIntoEditor(e: FileEntry) {
  selected.value = e
  loading.value = true
  error.value = ''
  try {
    const res = await filesApi.read(joinPath(cwd.value, e.name))
    editorContent.value = res.data.content
    dirty.value = false
    await mountEditor(e.name, editorContent.value)
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to read file'
  } finally {
    loading.value = false
  }
}

async function ensureMonaco() {
  if (monaco.value) return monaco.value
  monaco.value = await import('monaco-editor')
  return monaco.value
}

function languageForName(name: string): string {
  if (name.endsWith('.js') || name.endsWith('.mjs')) return 'javascript'
  if (name.endsWith('.ts')) return 'typescript'
  if (name.endsWith('.json')) return 'json'
  if (name.endsWith('.html') || name.endsWith('.htm')) return 'html'
  if (name.endsWith('.css')) return 'css'
  if (name.endsWith('.php')) return 'php'
  if (name.endsWith('.py')) return 'python'
  if (name.endsWith('.go')) return 'go'
  if (name.endsWith('.md')) return 'markdown'
  if (name.endsWith('.yml') || name.endsWith('.yaml')) return 'yaml'
  if (name.endsWith('.sh') || name.endsWith('.bash')) return 'shell'
  return 'plaintext'
}

async function mountEditor(name: string, content: string) {
  const m = await ensureMonaco()
  if (!editorContainer.value) return
  if (editorInstance) {
    editorInstance.dispose()
    editorInstance = null
  }
  editorInstance = m.editor.create(editorContainer.value, {
    value: content,
    language: languageForName(name),
    theme: 'vs',
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: 12,
    scrollBeyondLastLine: false,
  })
  editorInstance.onDidChangeModelContent(() => {
    editorContent.value = editorInstance.getValue()
    dirty.value = true
  })
}

function closeEditor() {
  if (editorInstance) {
    editorInstance.dispose()
    editorInstance = null
  }
  editorContent.value = ''
  dirty.value = false
}

async function save() {
  if (!selected.value || saving.value) return
  saving.value = true
  error.value = ''
  try {
    await filesApi.write(joinPath(cwd.value, selected.value.name), editorContent.value)
    dirty.value = false
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Save failed'
  } finally {
    saving.value = false
  }
}

async function createFile() {
  if (!newName.value) return
  await filesApi.write(joinPath(cwd.value, newName.value), '')
  showNewFile.value = false
  newName.value = ''
  await refresh()
}

async function createDir() {
  if (!newName.value) return
  await filesApi.mkdir(joinPath(cwd.value, newName.value))
  showNewDir.value = false
  newName.value = ''
  await refresh()
}

async function doRename() {
  if (!renameTarget.value || !renameNewName.value) return
  await filesApi.rename(
    joinPath(cwd.value, renameTarget.value.name),
    joinPath(cwd.value, renameNewName.value),
  )
  renameTarget.value = null
  renameNewName.value = ''
  await refresh()
}

async function doDelete() {
  if (!confirmDelete.value) return
  await filesApi.delete(joinPath(cwd.value, confirmDelete.value.name))
  if (selected.value && selected.value.name === confirmDelete.value.name) {
    selected.value = null
    closeEditor()
  }
  confirmDelete.value = null
  await refresh()
}

function formatSize(n: number) {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  if (n < 1024 * 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  return (n / 1024 / 1024 / 1024).toFixed(1) + ' GB'
}

watch(cwd, () => refresh())
onMounted(refresh)
</script>

<template>
  <div class="space-y-3 h-[calc(100vh-100px)] flex flex-col">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">File Manager</h1>
      <div class="flex gap-2">
        <button @click="showNewFile = true; newName = ''"
          class="text-xs bg-indigo-600 text-white px-3 py-1.5 rounded-md hover:bg-indigo-700">+ File</button>
        <button @click="showNewDir = true; newName = ''"
          class="text-xs bg-gray-100 text-gray-600 border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-200">+ Folder</button>
        <button @click="refresh"
          class="text-xs text-gray-600 border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-50">Refresh</button>
      </div>
    </div>

    <!-- Breadcrumb -->
    <div class="flex items-center gap-1 text-xs text-gray-500">
      <template v-for="(b, i) in breadcrumbs" :key="b.path">
        <button @click="cwd = b.path; selected = null; closeEditor()"
          class="hover:text-indigo-600">{{ b.label }}</button>
        <span v-if="i < breadcrumbs.length - 1" class="text-gray-300">/</span>
      </template>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">{{ error }}</p>

    <div class="flex-1 grid grid-cols-[280px_1fr] gap-3 min-h-0">
      <!-- File list -->
      <div class="bg-white border border-gray-200 rounded-lg overflow-y-auto">
        <div v-if="loading && !entries.length" class="p-3 text-xs text-gray-400">Loading...</div>
        <div v-else-if="!entries.length" class="p-3 text-xs text-gray-400">Empty directory</div>
        <ul v-else class="divide-y divide-gray-100">
          <li v-for="e in entries" :key="e.name"
            class="flex items-center justify-between px-3 py-2 text-xs hover:bg-gray-50 cursor-pointer"
            :class="selected?.name === e.name ? 'bg-indigo-50' : ''"
            @click="openEntry(e)">
            <div class="flex items-center gap-2 min-w-0">
              <svg v-if="e.is_dir" class="w-3.5 h-3.5 text-amber-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <svg v-else class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
              </svg>
              <span class="truncate" :class="e.is_dir ? 'text-gray-700 font-medium' : 'text-gray-600'">{{ e.name }}</span>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0 text-gray-400">
              <span v-if="!e.is_dir">{{ formatSize(e.size) }}</span>
              <button @click.stop="renameTarget = e; renameNewName = e.name" class="hover:text-indigo-600" title="Rename">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
              </button>
              <button @click.stop="confirmDelete = e" class="hover:text-red-500" title="Delete">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                </svg>
              </button>
            </div>
          </li>
        </ul>
      </div>

      <!-- Editor / placeholder -->
      <div class="bg-white border border-gray-200 rounded-lg flex flex-col min-h-0">
        <div v-if="!selected" class="flex-1 flex items-center justify-center text-xs text-gray-400">
          Select a file to edit
        </div>
        <template v-else>
          <div class="flex items-center justify-between px-3 py-2 border-b border-gray-100">
            <div class="text-xs text-gray-600 truncate">{{ joinPath(cwd, selected.name) }}</div>
            <div class="flex items-center gap-2">
              <span v-if="dirty" class="text-[10px] text-amber-600">Unsaved</span>
              <button @click="save" :disabled="!dirty || saving"
                class="text-xs bg-indigo-600 text-white px-3 py-1 rounded-md hover:bg-indigo-700 disabled:opacity-50">
                {{ saving ? 'Saving...' : 'Save' }}
              </button>
            </div>
          </div>
          <div ref="editorContainer" class="flex-1 min-h-0"></div>
        </template>
      </div>
    </div>

    <!-- New File modal -->
    <div v-if="showNewFile" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-sm shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-3">New File</h3>
        <input v-model="newName" placeholder="filename.txt"
          @keydown.enter="createFile"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <div class="flex gap-2 mt-4">
          <button @click="showNewFile = false" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="createFile" :disabled="!newName"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">Create</button>
        </div>
      </div>
    </div>

    <!-- New Folder modal -->
    <div v-if="showNewDir" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-sm shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-3">New Folder</h3>
        <input v-model="newName" placeholder="folder-name"
          @keydown.enter="createDir"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <div class="flex gap-2 mt-4">
          <button @click="showNewDir = false" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="createDir" :disabled="!newName"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">Create</button>
        </div>
      </div>
    </div>

    <!-- Rename modal -->
    <div v-if="renameTarget" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-sm shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-3">Rename "{{ renameTarget.name }}"</h3>
        <input v-model="renameNewName"
          @keydown.enter="doRename"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <div class="flex gap-2 mt-4">
          <button @click="renameTarget = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doRename" :disabled="!renameNewName || renameNewName === renameTarget.name"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">Rename</button>
        </div>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-sm shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-2">Delete "{{ confirmDelete.name }}"?</h3>
        <p class="text-xs text-gray-500 mb-4">
          {{ confirmDelete.is_dir ? 'This folder and everything inside will be permanently removed.' : 'This file will be permanently removed.' }}
        </p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doDelete" class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
