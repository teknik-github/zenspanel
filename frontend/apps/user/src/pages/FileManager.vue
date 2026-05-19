<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import { useRoute } from 'vue-router'
import { filesApi, type FileEntry } from '@/api/files'

const cwd = ref('')
const entries = ref<FileEntry[]>([])
const selected = ref<FileEntry | null>(null)
const editorContent = ref('')
const dirty = ref(false)
const loading = ref(false)
const saving = ref(false)
const error = ref('')

// Modals — only one is open at a time. Naming follows the action verb
// so the template reads naturally: showNewFile/Dir + the four below.
const showNewFile = ref(false)
const showNewDir = ref(false)
const newName = ref('')
const renameTarget = ref<FileEntry | null>(null)
const renameNewName = ref('')
const confirmDelete = ref<FileEntry | null>(null)

// Permissions modal — split octal into three rwx groups so the user can
// click checkboxes instead of typing octal. We sync octal ↔ checkboxes
// both ways so power users can paste 0755 and casual users can click.
const chmodTarget = ref<FileEntry | null>(null)
const chmodMode = ref('0644')
const chmodOwner = ref({ r: false, w: false, x: false })
const chmodGroup = ref({ r: false, w: false, x: false })
const chmodOther = ref({ r: false, w: false, x: false })

const copyTarget = ref<FileEntry | null>(null)
const copyDst = ref('')

const moveTarget = ref<FileEntry | null>(null)
const moveDst = ref('')

const compressTarget = ref<FileEntry | null>(null)
const compressDst = ref('')

const confirmExtract = ref<FileEntry | null>(null)

// Upload state. uploads holds in-flight files keyed by name; the list
// renders only as long as a file is in here, so completed uploads
// disappear from the toast strip naturally.
type UploadStatus = { name: string; pct: number; error?: string }
const uploads = ref<UploadStatus[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const isDragging = ref(false)

const textExtensions = new Set([
  'php', 'html', 'htm', 'css', 'js', 'mjs', 'ts', 'tsx', 'jsx',
  'json', 'md', 'txt', 'yaml', 'yml', 'sh', 'bash', 'env',
  'xml', 'csv', 'sql', 'ini', 'conf', 'toml', 'log', 'py', 'go',
])
const archiveExtensions = ['.zip', '.tar.gz', '.tgz']

function isTextFile(name: string): boolean {
  const dot = name.lastIndexOf('.')
  if (dot < 0) return true
  const ext = name.slice(dot + 1).toLowerCase()
  return textExtensions.has(ext)
}

function isArchive(name: string): boolean {
  const lower = name.toLowerCase()
  return archiveExtensions.some(ext => lower.endsWith(ext))
}

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
  selected.value = e
  if (!isTextFile(e.name)) {
    closeEditor()
    return
  }
  await loadFileIntoEditor(e)
}

async function loadFileIntoEditor(e: FileEntry) {
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

// === Permissions ============================================================

function openChmod(e: FileEntry) {
  chmodTarget.value = e
  // The mode column in the listing is the symbolic form (-rwxr-xr-x).
  // Convert it to the rwx checkboxes; octal is derived from those.
  const m = e.mode.length === 10 ? e.mode.slice(1) : e.mode
  if (m.length === 9) {
    chmodOwner.value = { r: m[0] === 'r', w: m[1] === 'w', x: m[2] === 'x' }
    chmodGroup.value = { r: m[3] === 'r', w: m[4] === 'w', x: m[5] === 'x' }
    chmodOther.value = { r: m[6] === 'r', w: m[7] === 'w', x: m[8] === 'x' }
  } else {
    chmodOwner.value = { r: true, w: true, x: false }
    chmodGroup.value = { r: true, w: false, x: false }
    chmodOther.value = { r: true, w: false, x: false }
  }
  chmodMode.value = computeOctal()
}

function computeOctal(): string {
  const bit = (b: { r: boolean; w: boolean; x: boolean }) =>
    (b.r ? 4 : 0) + (b.w ? 2 : 0) + (b.x ? 1 : 0)
  return '0' + bit(chmodOwner.value).toString() +
    bit(chmodGroup.value).toString() +
    bit(chmodOther.value).toString()
}

watch([chmodOwner, chmodGroup, chmodOther], () => {
  chmodMode.value = computeOctal()
}, { deep: true })

async function applyChmod() {
  if (!chmodTarget.value) return
  await filesApi.chmod(joinPath(cwd.value, chmodTarget.value.name), chmodMode.value)
  chmodTarget.value = null
  await refresh()
}

// === Copy / Move ===========================================================

function openCopy(e: FileEntry) {
  copyTarget.value = e
  copyDst.value = joinPath(cwd.value, 'copy-of-' + e.name)
}

async function doCopy() {
  if (!copyTarget.value || !copyDst.value) return
  await filesApi.copy(joinPath(cwd.value, copyTarget.value.name), copyDst.value)
  copyTarget.value = null
  copyDst.value = ''
  await refresh()
}

function openMove(e: FileEntry) {
  moveTarget.value = e
  moveDst.value = joinPath(cwd.value, e.name)
}

async function doMove() {
  if (!moveTarget.value || !moveDst.value) return
  await filesApi.rename(joinPath(cwd.value, moveTarget.value.name), moveDst.value)
  moveTarget.value = null
  moveDst.value = ''
  await refresh()
}

// === Compress / Extract ====================================================

function openCompress(e: FileEntry) {
  compressTarget.value = e
  compressDst.value = joinPath(cwd.value, e.name + '.zip')
}

async function doCompress() {
  if (!compressTarget.value || !compressDst.value) return
  loading.value = true
  try {
    await filesApi.compress(joinPath(cwd.value, compressTarget.value.name), compressDst.value)
    compressTarget.value = null
    compressDst.value = ''
    await refresh()
  } finally {
    loading.value = false
  }
}

async function doExtract() {
  if (!confirmExtract.value) return
  loading.value = true
  try {
    await filesApi.extract(joinPath(cwd.value, confirmExtract.value.name), cwd.value)
    confirmExtract.value = null
    await refresh()
  } finally {
    loading.value = false
  }
}

// === Upload (existing) =====================================================

async function uploadFiles(files: FileList | File[]) {
  const list = Array.from(files)
  for (const f of list) {
    const status: UploadStatus = { name: f.name, pct: 0 }
    uploads.value.push(status)
    try {
      await filesApi.upload(cwd.value, f, pct => { status.pct = pct })
      status.pct = 100
    } catch (e: any) {
      status.error = e.response?.data?.error || 'upload failed'
    }
    setTimeout(() => {
      uploads.value = uploads.value.filter(u => u !== status)
    }, status.error ? 4000 : 1500)
  }
  await refresh()
}

function triggerFilePicker() {
  fileInput.value?.click()
}

function onFileInputChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length) {
    uploadFiles(input.files)
    input.value = ''
  }
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  isDragging.value = true
}

function onDragLeave() {
  isDragging.value = false
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false
  if (e.dataTransfer?.files?.length) {
    uploadFiles(e.dataTransfer.files)
  }
}

function formatSize(n: number) {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  if (n < 1024 * 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  return (n / 1024 / 1024 / 1024).toFixed(1) + ' GB'
}

watch(cwd, () => refresh())
onMounted(() => {
  const route = useRoute()
  const queryPath = route.query.path
  if (typeof queryPath === 'string' && queryPath !== '') {
    cwd.value = queryPath
  }
  refresh()
})

onUnmounted(() => closeEditor())
</script>

<template>
  <div class="space-y-3 h-[calc(100vh-100px)] flex flex-col">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">File Manager</h1>
      <div class="flex gap-2">
        <input ref="fileInput" type="file" multiple class="hidden" @change="onFileInputChange" />
        <button @click="triggerFilePicker" title="Upload files"
          class="text-xs bg-indigo-600 text-white px-3 py-1.5 rounded-md hover:bg-indigo-700 inline-flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
          </svg>
          Upload
        </button>
        <button @click="showNewFile = true; newName = ''" title="New file"
          class="text-xs bg-gray-100 text-gray-600 border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-200 inline-flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/>
          </svg>
          File
        </button>
        <button @click="showNewDir = true; newName = ''" title="New folder"
          class="text-xs bg-gray-100 text-gray-600 border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-200 inline-flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>
          </svg>
          Folder
        </button>
        <button @click="refresh" title="Refresh"
          class="text-xs text-gray-600 border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-50 inline-flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
          Refresh
        </button>
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

    <div class="flex-1 grid grid-cols-[320px_1fr] gap-3 min-h-0">
      <!-- File list -->
      <div class="bg-white border border-gray-200 rounded-lg overflow-y-auto relative"
        :class="{ 'ring-2 ring-indigo-300': isDragging }"
        @dragover="onDragOver" @dragleave="onDragLeave" @drop="onDrop">
        <div v-if="isDragging" class="absolute inset-0 bg-indigo-50/80 flex items-center justify-center pointer-events-none z-10 text-xs text-indigo-700 font-medium">
          Drop files to upload
        </div>
        <div v-if="loading && !entries.length" class="p-3 text-xs text-gray-400">Loading...</div>
        <div v-else-if="!entries.length" class="p-3 text-xs text-gray-400">Empty directory</div>
        <ul v-else class="divide-y divide-gray-100">
          <li v-for="e in entries" :key="e.name"
            class="flex items-center justify-between px-3 py-2 text-xs hover:bg-gray-50 cursor-pointer"
            :class="selected?.name === e.name ? 'bg-indigo-50' : ''"
            @click="openEntry(e)">
            <div class="flex items-center gap-2 min-w-0 flex-1">
              <svg v-if="e.is_dir" class="w-3.5 h-3.5 text-amber-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <svg v-else class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
              </svg>
              <span class="truncate" :class="e.is_dir ? 'text-gray-700 font-medium' : 'text-gray-600'">{{ e.name }}</span>
            </div>
            <div class="flex items-center gap-1 flex-shrink-0 text-gray-400 ml-2">
              <span v-if="!e.is_dir" class="text-[10px] mr-1">{{ formatSize(e.size) }}</span>
              <button v-if="isArchive(e.name)" @click.stop="confirmExtract = e" class="hover:text-indigo-600 p-1" title="Extract">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 8v13H3V8"/><rect x="1" y="3" width="22" height="5"/><line x1="10" y1="12" x2="14" y2="12"/>
                </svg>
              </button>
              <button @click.stop="openCompress(e)" class="hover:text-indigo-600 p-1" title="Compress">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 8v13H3V8"/><rect x="1" y="3" width="22" height="5"/><path d="M10 12h4"/>
                </svg>
              </button>
              <button @click.stop="openCopy(e)" class="hover:text-indigo-600 p-1" title="Copy">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                </svg>
              </button>
              <button @click.stop="openMove(e)" class="hover:text-indigo-600 p-1" title="Move">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="5 9 2 12 5 15"/><polyline points="9 5 12 2 15 5"/><polyline points="15 19 12 22 9 19"/><polyline points="19 9 22 12 19 15"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="12" y1="2" x2="12" y2="22"/>
                </svg>
              </button>
              <button @click.stop="openChmod(e)" class="hover:text-indigo-600 p-1" title="Permissions">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
              </button>
              <button @click.stop="renameTarget = e; renameNewName = e.name" class="hover:text-indigo-600 p-1" title="Rename">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
              </button>
              <button @click.stop="confirmDelete = e" class="hover:text-red-500 p-1" title="Delete">
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
        <template v-else-if="!isTextFile(selected.name)">
          <div class="flex items-center justify-between px-3 py-2 border-b border-gray-100">
            <div class="text-xs text-gray-600 truncate">{{ joinPath(cwd, selected.name) }}</div>
          </div>
          <div class="flex-1 flex items-center justify-center text-xs text-gray-400 p-6 text-center">
            Binary file — cannot be edited in the browser.<br>
            Use Download or Extract instead.
          </div>
        </template>
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

    <!-- Upload progress strip -->
    <div v-if="uploads.length" class="fixed bottom-4 right-4 space-y-1 z-50 w-72">
      <div v-for="u in uploads" :key="u.name"
        class="bg-white border border-gray-200 rounded-md shadow-md px-3 py-2 text-xs">
        <div class="flex items-center justify-between mb-1">
          <span class="truncate font-medium text-gray-700">{{ u.name }}</span>
          <span class="text-gray-400">{{ u.error ? 'error' : u.pct + '%' }}</span>
        </div>
        <div v-if="!u.error" class="h-1 bg-gray-100 rounded">
          <div class="h-1 bg-indigo-500 rounded transition-all" :style="{ width: u.pct + '%' }"></div>
        </div>
        <div v-else class="text-red-600">{{ u.error }}</div>
      </div>
    </div>

    <!-- New File modal -->
    <div v-if="showNewFile" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-sm shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-3">New File</h3>
        <input v-model="newName" placeholder="filename.txt" @keydown.enter="createFile"
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
        <input v-model="newName" placeholder="folder-name" @keydown.enter="createDir"
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
        <input v-model="renameNewName" @keydown.enter="doRename"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <div class="flex gap-2 mt-4">
          <button @click="renameTarget = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doRename" :disabled="!renameNewName || renameNewName === renameTarget.name"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">Rename</button>
        </div>
      </div>
    </div>

    <!-- Permissions modal -->
    <div v-if="chmodTarget" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-md shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-1">Permissions: {{ chmodTarget.name }}</h3>
        <p class="text-[11px] text-gray-400 mb-3">Click checkboxes or type the octal mode directly.</p>
        <table class="w-full text-xs mb-3">
          <thead>
            <tr class="text-gray-500 border-b border-gray-100">
              <th class="text-left pb-1 font-medium"></th>
              <th class="font-medium">Read</th>
              <th class="font-medium">Write</th>
              <th class="font-medium">Execute</th>
            </tr>
          </thead>
          <tbody>
            <tr class="border-b border-gray-50">
              <td class="py-2 text-gray-500">Owner</td>
              <td class="text-center"><input type="checkbox" v-model="chmodOwner.r" /></td>
              <td class="text-center"><input type="checkbox" v-model="chmodOwner.w" /></td>
              <td class="text-center"><input type="checkbox" v-model="chmodOwner.x" /></td>
            </tr>
            <tr class="border-b border-gray-50">
              <td class="py-2 text-gray-500">Group</td>
              <td class="text-center"><input type="checkbox" v-model="chmodGroup.r" /></td>
              <td class="text-center"><input type="checkbox" v-model="chmodGroup.w" /></td>
              <td class="text-center"><input type="checkbox" v-model="chmodGroup.x" /></td>
            </tr>
            <tr>
              <td class="py-2 text-gray-500">Other</td>
              <td class="text-center"><input type="checkbox" v-model="chmodOther.r" /></td>
              <td class="text-center"><input type="checkbox" v-model="chmodOther.w" /></td>
              <td class="text-center"><input type="checkbox" v-model="chmodOther.x" /></td>
            </tr>
          </tbody>
        </table>
        <div class="flex items-center gap-2 mb-4">
          <label class="text-xs text-gray-500">Octal:</label>
          <input v-model="chmodMode" class="border border-gray-200 rounded-md px-2 py-1 text-sm w-24 font-mono" />
          <span class="text-[10px] text-gray-400">e.g. 0755 (rwxr-xr-x)</span>
        </div>
        <div class="flex gap-2">
          <button @click="chmodTarget = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="applyChmod" class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700">Apply</button>
        </div>
      </div>
    </div>

    <!-- Copy modal -->
    <div v-if="copyTarget" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-md shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-3">Copy "{{ copyTarget.name }}"</h3>
        <label class="block text-xs text-gray-500 mb-1">Destination path</label>
        <input v-model="copyDst" @keydown.enter="doCopy"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <div class="flex gap-2 mt-4">
          <button @click="copyTarget = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doCopy" :disabled="!copyDst"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">Copy</button>
        </div>
      </div>
    </div>

    <!-- Move modal -->
    <div v-if="moveTarget" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-md shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-3">Move "{{ moveTarget.name }}"</h3>
        <label class="block text-xs text-gray-500 mb-1">New path (full destination including filename)</label>
        <input v-model="moveDst" @keydown.enter="doMove"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <div class="flex gap-2 mt-4">
          <button @click="moveTarget = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doMove" :disabled="!moveDst"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">Move</button>
        </div>
      </div>
    </div>

    <!-- Compress modal -->
    <div v-if="compressTarget" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-md shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-3">Compress "{{ compressTarget.name }}"</h3>
        <label class="block text-xs text-gray-500 mb-1">Output archive (.zip or .tar.gz)</label>
        <input v-model="compressDst" @keydown.enter="doCompress"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        <div class="flex gap-2 mt-4">
          <button @click="compressTarget = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doCompress" :disabled="!compressDst || loading"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Compressing...' : 'Compress' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Extract confirm -->
    <div v-if="confirmExtract" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-5 w-full max-w-sm shadow-xl">
        <h3 class="text-sm font-semibold text-gray-800 mb-2">Extract "{{ confirmExtract.name }}"?</h3>
        <p class="text-xs text-gray-500 mb-4">Files will be extracted into the current directory. Existing files with the same name will be overwritten.</p>
        <div class="flex gap-2">
          <button @click="confirmExtract = null" class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doExtract" :disabled="loading"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Extracting...' : 'Extract' }}
          </button>
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
