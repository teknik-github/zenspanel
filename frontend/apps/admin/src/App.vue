<script setup lang="ts">
import AppNotify from './AppNotify.vue'
</script>

<template>
  <router-view v-slot="{ Component }">
    <Transition name="page" mode="out-in">
      <component :is="Component" />
    </Transition>
  </router-view>
  <AppNotify />
</template>
