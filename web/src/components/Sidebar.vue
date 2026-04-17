<template>
  <aside class="sidebar">
    <!-- Logo / Header -->
    <div class="sidebar-header">
      <span class="sidebar-title">NTS Gater</span>
    </div>

    <!-- Menu -->
    <nav class="sidebar-nav">
      <div class="menu-group">
        <div
          class="menu-item"
          :class="{ active: isExpanded }"
          @click="toggleExpand"
        >
          <span>协议测试</span>
          <el-icon class="menu-arrow" :class="{ expanded: isExpanded }">
            <ArrowDown />
          </el-icon>
        </div>

        <transition name="slide">
          <div v-if="isExpanded" class="submenu">
            <div
              v-for="item in submenuItems"
              :key="item.key"
              class="submenu-item"
              :class="{ active: activeSubmenu === item.key }"
              @click="$emit('select-submenu', item.key)"
            >
              {{ item.label }}
            </div>
          </div>
        </transition>
      </div>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'

defineEmits<{
  'select-submenu': [key: string]
}>()

const isExpanded = ref(true)
const activeSubmenu = ref('standard')

const submenuItems = [
  { key: 'standard', label: '标准协议' },
]

function toggleExpand() {
  isExpanded.value = !isExpanded.value
}
</script>

<style scoped>
.sidebar {
  width: 200px;
  min-width: 200px;
  height: 100vh;
  background-color: #2B8A9A;
  display: flex;
  flex-direction: column;
  color: #fff;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 24px 20px 16px;
}

.sidebar-title {
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 1px;
}

.sidebar-nav {
  flex: 1;
  padding: 8px 0;
}

.menu-group {
  padding: 4px 10px;
}

.menu-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  cursor: pointer;
  border-radius: 6px;
  font-size: 14px;
  transition: background-color 0.2s;
}

.menu-item:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.menu-item.active {
  background-color: rgba(255, 255, 255, 0.15);
}

.menu-arrow {
  font-size: 12px;
  transition: transform 0.3s ease;
  color: rgba(255, 255, 255, 0.7);
}

.menu-arrow.expanded {
  transform: rotate(180deg);
}

.submenu {
  margin-top: 4px;
  margin-left: 8px;
  overflow: hidden;
}

.submenu-item {
  padding: 9px 14px;
  font-size: 13px;
  cursor: pointer;
  border-radius: 5px;
  color: rgba(255, 255, 255, 0.75);
  transition: all 0.2s;
}

.submenu-item:hover {
  background-color: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.submenu-item.active {
  color: #fff;
  background-color: rgba(255, 255, 255, 0.15);
  font-weight: 500;
}

/* Submenu animation */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.25s ease;
  max-height: 300px;
}
.slide-enter-from,
.slide-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
