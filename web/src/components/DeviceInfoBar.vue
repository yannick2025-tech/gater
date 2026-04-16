<template>
  <div class="bg-white p-4 rounded-lg shadow mb-6 flex items-center justify-between">
    <div class="flex items-center gap-6">
      <div>
        <span class="text-gray-500 mr-2">枪编号:</span>
        <span class="font-medium">{{ gunNumber || '-' }}</span>
      </div>
      <div>
        <span class="text-gray-500 mr-2">协议:</span>
        <span class="font-medium">{{ protocolName }} {{ protocolVersion }}</span>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-gray-500">状态:</span>
        <span
          :class="[
            'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
            isOnline ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
          ]"
        >
          <span :class="['w-2 h-2 rounded-full', isOnline ? 'bg-green-500' : 'bg-gray-400']" />
          {{ isOnline ? '在线' : '离线' }}
        </span>
      </div>
    </div>

    <div class="flex items-center gap-3">
      <el-input
        v-model="inputGunNumber"
        placeholder="输入枪编号"
        class="w-48"
        size="default"
        @keyup.enter="handleQuery"
      />
      <el-button
        type="primary"
        :loading="loading"
        @click="handleQuery"
        style="background-color: #148493; border-color: #148493"
      >
        查询
      </el-button>
      <el-button
        v-if="isOnline"
        type="danger"
        :loading="loading"
        @click="$emit('disconnect')"
      >
        断开
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  gunNumber: string
  protocolName: string
  protocolVersion: string
  isOnline: boolean
  loading: boolean
}>()

const emit = defineEmits<{
  disconnect: []
  query: [gunNumber: string]
}>()

const inputGunNumber = ref(props.gunNumber)

function handleQuery() {
  if (inputGunNumber.value.trim()) {
    emit('query', inputGunNumber.value.trim())
  }
}
</script>
