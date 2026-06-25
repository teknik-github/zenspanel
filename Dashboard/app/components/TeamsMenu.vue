<script setup lang="ts">
defineProps<{
  collapsed?: boolean
}>()
</script>

<template>
  <div class="flex items-center gap-2 px-2 py-2">
    <UIcon name="i-lucide-zap" class="size-6 shrink-0 text-primary" />
    <span
      v-if="!collapsed"
      class="font-semibold text-lg truncate"
    >ZensPanel</span>
  </div>
</template>
