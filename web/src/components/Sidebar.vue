<template>
  <div class="w-64 h-screen flex flex-col" style="background-color: #148493">
    <div class="p-6">
      <h2 class="text-white text-lg font-semibold mb-6">系统菜单</h2>

      <nav>
        <div>
          <button
            class="w-full flex items-center justify-between px-4 py-3 text-white hover:bg-[#0E6E7B] rounded transition-colors"
            @click="isProtocolExpanded = !isProtocolExpanded"
          >
            <span>协议测试</span>
            <ChevronDown
              :class="['w-4 h-4 transition-transform', isProtocolExpanded ? 'rotate-180' : '']"
            />
          </button>

          <div v-if="isProtocolExpanded" class="ml-4 mt-1">
            <router-link
              to="/"
              class="block w-full text-left px-4 py-2 text-white rounded transition-colors"
              style="background-color: #0E6E7B"
            >
              XX标准协议
            </router-link>
          </div>
        </div>
      </nav>
    </div>

    <div class="mt-auto p-6">
      <div class="text-white/60 text-xs">
        NTS-Gater v1.0.0
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ChevronDown } from 'lucide-vue-next'

const isProtocolExpanded = ref(true)
</script>
