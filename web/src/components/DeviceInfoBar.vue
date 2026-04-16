<template>
  <div class="device-info-bar">
    <div class="info-left">
      <span class="info-item">
        <label>检编号：</label>{{ gunNumber || '--' }}
      </span>
      <div class="info-divider"></div>
      <span class="info-item">
        <label>协议名称：</label>{{ protocolName || 'XX标准协议' }}
      </span>
      <div class="info-divider"></div>
      <span class="info-item">
        <label>协议版本：</label>{{ protocolVersion || 'v1.6.0' }}
      </span>
    </div>
    <div class="info-right">
      <!-- 未连接状态：显示输入框+连接按钮 -->
      <template v-if="!isOnline">
        <el-input
          v-model="inputGunNumber"
          placeholder="请输入充电桩编号"
          clearable
          size="default"
          class="gun-input"
          @keyup.enter="handleConnect"
        >
          <template #append>
            <el-button type="success" :disabled="!inputGunNumber.trim()" @click="handleConnect">
              连接设备
            </el-button>
          </template>
        </el-input>
      </template>

      <!-- 已连接状态：显示断开按钮 -->
      <template v-else>
        <el-button
          type="danger"
          class="disconnect-btn"
          @click="$emit('disconnect')"
        >
          断开连接
        </el-button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  gunNumber: string
  isOnline?: boolean
  protocolName?: string
  protocolVersion?: string
}>()

const emit = defineEmits<{
  connect: [gunNumber: string]
  disconnect: []
  query: [gunNumber: string]
}>()

const inputGunNumber = ref('')

// 外部枪号变化时同步到输入框（如通过query触发）
watch(() => props.gunNumber, (val) => {
  if (val && !inputGunNumber.value) {
    inputGunNumber.value = val
  }
})

function handleConnect() {
  const num = inputGunNumber.value.trim()
  if (num) {
    emit('connect', num)
  }
}
</script>

<style scoped>
.device-info-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-radius: 8px;
  padding: 16px 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.info-left {
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-item {
  font-size: 14px;
  color: #333;
  white-space: nowrap;
}

.info-item label {
  color: #999;
  margin-right: 2px;
}

.info-divider {
  width: 1px;
  height: 14px;
  background-color: #ddd;
  margin: 0 12px;
}

.info-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.gun-input {
  width: 320px;
}

.disconnect-btn {
  border-radius: 6px;
  padding: 8px 20px;
  font-size: 13px;
}
</style>
