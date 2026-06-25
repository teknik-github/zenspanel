<script setup lang="ts">
const toast = useToast()
const currentPath = ref('')
const files = ref<any[]>([])
const loading = ref(false)

async function listFiles(path?: string) {
  loading.value = true
  currentPath.value = path || ''
  try {
    const res: any = await $fetch('/api/v1/files', { query: { path: path || '' } })
    files.value = res?.data || []
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
  finally { loading.value = false }
}

onMounted(() => listFiles())

function isDir(f: any) { return f.is_dir }
function navigateTo(f: any) {
  if (!f.is_dir) return
  const next = currentPath.value ? `${currentPath.value}/${f.name}` : f.name
  listFiles(next)
}
function goUp() {
  if (!currentPath.value) return
  const parts = currentPath.value.split('/'); parts.pop()
  listFiles(parts.join('/'))
}

// Create directory
const mkdirName = ref(''); const mkdirOpen = ref(false); const mkdirLoading = ref(false)
async function handleMkdir() {
  mkdirLoading.value = true
  try {
    await $fetch('/api/v1/files/mkdir', { method: 'POST', body: { path: `${currentPath.value}/${mkdirName.value}` } })
    toast.add({ title: 'Directory created', color: 'success' }); mkdirOpen.value = false; mkdirName.value = ''; listFiles(currentPath.value)
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
  finally { mkdirLoading.value = false }
}

// Delete
async function handleDelete(f: any) {
  const fullPath = currentPath.value ? `${currentPath.value}/${f.name}` : f.name
  if (!confirm(`Delete ${fullPath}? ${f.is_dir ? 'Recursive!' : ''}`)) return
  try {
    await $fetch('/api/v1/files', { method: 'DELETE', query: { path: fullPath } })
    toast.add({ title: 'Deleted', color: 'success' }); listFiles(currentPath.value)
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
}

// Upload
const uploadInput = ref<HTMLInputElement>()
async function handleUpload(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  const form = new FormData()
  form.append('path', currentPath.value)
  form.append('file', input.files[0] as File)
  try {
    await $fetch('/api/v1/files/upload', { method: 'POST', body: form as unknown as Record<string, any> })
    toast.add({ title: 'Uploaded', color: 'success' }); listFiles(currentPath.value)
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
}

function fmtSize(b: number) { if (!b) return '—'; if (b >= 1048576) return (b/1048576).toFixed(1)+' MB'; return (b/1024).toFixed(0)+' KB' }
</script>

<template>
<UDashboardPanel id="file-manager"><template #header><UDashboardNavbar title="File Manager"><template #leading><UDashboardSidebarCollapse /></template><template #right><UButton label="New Folder" icon="i-lucide-folder-plus" variant="outline" @click="mkdirOpen=true" /><UButton label="Upload" icon="i-lucide-upload" class="ml-2" @click="uploadInput?.click()" /><input ref="uploadInput" type="file" hidden @change="handleUpload" /></template></UDashboardNavbar></template>
<template #body>
<div class="flex items-center gap-2 mb-4">
  <UButton icon="i-lucide-arrow-left" size="xs" color="neutral" variant="ghost" @click="goUp" :disabled="!currentPath" />
  <code class="text-sm font-mono text-highlighted">~/{{ currentPath || '' }}</code>
</div>
<div v-if="loading" class="text-dimmed text-sm">Loading...</div>
<div v-else class="space-y-0.5">
  <div v-for="f in files" :key="f.name" class="flex items-center justify-between p-2 rounded-lg hover:bg-elevated/50 cursor-pointer" @click="navigateTo(f)">
    <div class="flex items-center gap-3">
      <UIcon :name="f.is_dir?'i-lucide-folder':'i-lucide-file'" class="size-4" :class="f.is_dir?'text-primary':''" />
      <span class="text-sm" :class="f.is_dir?'font-medium text-highlighted':''">{{ f.name }}</span>
    </div>
    <div class="flex items-center gap-3">
      <span class="text-xs text-dimmed">{{ f.is_dir ? '—' : fmtSize(f.size) }}</span>
      <span class="text-xs text-dimmed font-mono">{{ f.mode }}</span>
      <UButton icon="i-lucide-trash" size="xs" color="neutral" variant="ghost" @click.stop="handleDelete(f)" />
    </div>
  </div>
  <div v-if="!files.length" class="text-sm text-dimmed text-center py-8">Empty directory</div>
</div>

<UModal v-model:open="mkdirOpen" title="New Folder"><template #body>
<div class="space-y-4">
  <UFormField label="Folder name"><UInput v-model="mkdirName" :disabled="mkdirLoading" placeholder="uploads" /></UFormField>
  <div class="flex justify-end gap-2"><UButton label="Cancel" color="neutral" variant="subtle" @click="mkdirOpen=false" :disabled="mkdirLoading" /><UButton label="Create" :loading="mkdirLoading" @click="handleMkdir" /></div>
</div></template></UModal>
</template></UDashboardPanel>
</template>
