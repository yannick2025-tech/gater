<template>
  <div class="device-info-bar">
    <div class="info-left">
      <span class="info-item">
        <label>桩编号：</label>{{ gunNumber || '--' }}
      </span>
      <div class="info-divider"></div>
      <span class="info-item">
        <label>协议名称：</label>{{ protocolName || 'XX标准协议' }}
      </span>
      <div class="info-divider"></div>
      <span class="info-item">
        <label>协议版本：</label>{{ protocolVersion || 'v1.6.0' }}
      </span>
      <div class="info-divider"></div>
      <el-tag v-if="isOnline" type="success" size="small" effect="light" round>在线</el-tag>
      <el-tag v-if="!isOnline && gunNumber" type="info" size="small" effect="light" round>离线</el-tag>
    </div>

    <div class="info-right">
      <!-- 会话选择器：下拉选择活跃会话 -->
      <template v-if="!isOnline">
        <el-select
          v-model="selectedSessionId"
          placeholder="请选择充电桩会话"
          clearable
          filterable
          size="default"
          class="session-select"
          @change="handleSessionSelect"
        >
          <el-option-group v-for="(group, label) in groupedSessions" :key="label" :label="label">
            <el-option
              v-for="s in group"
              :key="s.sessionId"
              :value="s.sessionId"
              :label="`桩${s.gunNumber} (${s.connectedAt})`"
              :disabled="!s.isOnline"
            >
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <span>桩{{ s.gunNumber }}</span>
                <el-tag v-if="s.isOnline" type="success" size="small">在线</el-tag>
                <el-tag v-else type="info" size="small">{{ s.authState }}</el-tag>
              </div>
            </el-option>
          </el-option-group>
        </el-select>
        <el-button plain size="default" class="refresh-btn" @click="$emit('refresh-sessions')">
          刷新列表
        </el-button>
      </template>

      <!-- 已选中会话：显示断开按钮（历史会话置灰） -->
      <template v-else>
        <el-button
          type="danger"
          class="disconnect-btn"
          :disabled="isHistorical"
          :plain="isHistorical"
          @click="$emit('disconnect')"
        >
          {{ isHistorical ? '历史会话' : '断开连接' }}
        </el-button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SessionItem } from '@/types/device'

const props = defineProps<{
  gunNumber: string
  isOnline?: boolean
  protocolName?: string
  protocolVersion?: string
  sessions?: SessionItem[]
}>()

const emit = defineEmits<{
  'select-session': [session: SessionItem | null]
  disconnect: []
  'refresh-sessions': []
}>()

const selectedSessionId = ref<string>('')

// 将session按状态分组：在线的在前，认证中的其次，其他的最后
const groupedSessions = computed(() => {
  const list = props.sessions || []
  const online = list.filter(s => s.isOnline)
  const pending = list.filter(s => !s.isOnline && s.authState === 'pending')
  const offline = list.filter(s => !s.isOnline && s.authState !== 'pending')
  
  const result: Record<string, SessionItem[]> = {}
  if (online.length > 0) result['在线会话'] = online
  if (pending.length > 0) result['认证中'] = pending
  if (offline.length > 0) result['离线/其他'] = offline
  return result
})

// 是否为历史会话（已断开但还在内存或DB中有记录）
const isHistorical = computed(() => {
  return !props.isOnline && !!props.gunNumber
})

// 外部props变化时同步selectedSessionId
function syncFromProps() {
  if (!props.isOnline && selectedSessionId.value) {
    // 断开了，清空选择
    selectedSessionId.value = ''
  }
}

function handleSessionSelect(sessionId: string | undefined) {
  if (!sessionId) {
    emit('select-session', null)
    return
  }
  const sess = props.sessions?.find(s => s.sessionId === sessionId)
  emit('select-session', sess || null)
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

.session-select {
  width: 320px;
}

.refresh-btn {
  border-radius: 6px;
}

.disconnect-btn {
  border-radius: 6px;
  padding: 8px 20px;
  font-size: 13px;
}
</style>
