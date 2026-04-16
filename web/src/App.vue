<template>
  <div class="flex h-screen overflow-hidden">
    <Sidebar />
    <router-view />
  </div>
</template>

<script setup lang="ts">
import Sidebar from '@/components/Sidebar.vue'
</script>
