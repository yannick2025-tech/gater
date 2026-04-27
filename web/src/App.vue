<template>
  <div class="app-layout">
    <Sidebar @select-submenu="handleSelectSubmenu" />
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import Sidebar from '@/components/Sidebar.vue'

const router = useRouter()

/** 点击侧边栏二级菜单 → 跳转到首页（显示所有历史TCP会话列表） */
function handleSelectSubmenu(_key: string) {
  router.push({ name: 'ProtocolTest' })
}
</script>

<style>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.main-content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  background-color: #f5f6fa;
}
</style>
