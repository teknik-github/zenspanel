<script setup lang="ts">
const auth = useAuth()
const target = auth.user.value?.role === 'admin' ? '/admin/dashboard' : '/dashboard'

await navigateTo(target)
</script>

<template>
  <div class="min-h-screen flex items-center justify-center">
    <p class="text-dimmed">
      Redirecting...
    </p>
  </div>
</template>
