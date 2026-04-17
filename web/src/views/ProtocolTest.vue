<template>
  <div class="protocol-test-page">
    <!-- 第1块：会话选择 / 设备信息（双模式） -->
    <DeviceInfoBar
      :gun-number="deviceStore.deviceInfo.gunNumber"
      :is-online="deviceStore.deviceInfo.isOnline"
      :protocol-name="deviceStore.deviceInfo.protocolName"
      :protocol-version="deviceStore.deviceInfo.protocolVersion"
      :sessions="deviceStore.sessionList"
      :mode="viewMode"
      @select-session="handleSelectSession"
      @disconnect="handleDisconnect"
      @refresh-sessions="deviceStore.fetchSessions"
      @back-to-list="handleBackToList"
    />

    <!-- 第2块：测试配置（仅活跃会话时显示，否则隐藏占位） -->
    <div class="section-wrapper" :class="{ 'section-hidden': !showTestConfig }">
      <TestConfig
        v-if="showTestConfig"
        :can-start-test="hasActiveSession"
        @start="handleStartTest"
      />
    </div>

    <!-- 第3块：测试结果（选中会话后显示，否则隐藏占位） -->
    <div class="section-wrapper" :class="{ 'section-hidden': !showTestResults }">
      <TestResults
        v-if="showTestResults"
        :results="filteredResults"
        @view-detail="handleViewDetail"
        @export="handleExportReport"
      />
    </div>

    <!-- Detail Modal -->
    <TestDetailModal
      v-model="showDetailModal"
      :session-id="selectedSessionId"
      @view-messages="handleViewMessages"
    />

    <!-- Message View Modal -->
    <MessageViewModal v-model="showMessageModal" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useDeviceStore } from '@/stores/device'
import { useTestStore } from '@/stores/test'
import type { SessionItem } from '@/types/device'
import DeviceInfoBar from '@/components/DeviceInfoBar.vue'
import TestConfig from '@/components/TestConfig.vue'
import TestResults from '@/components/TestResults.vue'
import TestDetailModal from '@/components/TestDetailModal.vue'
import MessageViewModal from '@/components/MessageViewModal.vue'

const deviceStore = useDeviceStore()
const testStore = useTestStore()

const showDetailModal = ref(false)
const showMessageModal = ref(false)
const selectedTestId = ref<number | null>(null)
const selectedSessionId = ref<string>('')

// ========== 三态逻辑：none / active / historical ==========

type SelectionState = 'none' | 'active' | 'historical'

/** 当前选择状态 */
const selectionState = computed<SelectionState>(() => {
  const sess = deviceStore.selectedSession
  if (!sess) return 'none'
  if (sess.isOnline) return 'active'
  return 'historical'
})

/** DeviceInfoBar 的显示模式 */
const viewMode = computed<'select' | 'detail'>(() => {
  return selectionState.value === 'none' ? 'select' : 'detail'
})

/** 是否有活跃会话（用于控制"开始测试"按钮） */
const hasActiveSession = computed(() => selectionState.value === 'active')

/** 是否显示测试配置区B */
const showTestConfig = computed(() => selectionState.value === 'active')

/** 是否显示测试结果区C */
const showTestResults = computed(() => selectionState.value !== 'none')

/**
 * 测试结果列表：
 * - 活跃会话：显示全部结果（含运行中的）
 * - 历史会话：仅显示该session的结果
 */
const filteredResults = computed(() => {
  const sess = deviceStore.selectedSession
  if (!sess) return []
  if (selectionState.value === 'active') {
    // 活跃会话：返回全部结果
    return testStore.testResults
  }
  // 历史会话：仅过滤该sessionId的记录
  return testStore.testResults.filter(r => r.sessionId === sess.sessionId)
})

// 页面挂载时启动会话列表轮询
onMounted(() => {
  deviceStore.startSessionPolling(10000)
})

// 页面卸载时停止所有轮询
onUnmounted(() => {
  deviceStore.stopSessionPolling()
  testStore.stopPolling()
})

// ========== 事件处理 ==========

function handleSelectSession(session: SessionItem) {
  deviceStore.selectSession(session)
  // 选中历史会话时，加载该会话的测试结果
  if (!session.isOnline) {
    testStore.fetchResultsBySession(session.sessionId)
  }
}

function handleBackToList() {
  deviceStore.reset()
}

function handleDisconnect() {
  // 1. 先停止轮询，标记测试为已完成
  if (testStore.currentStatus?.sessionId) {
    testStore.markTestCompleted(testStore.currentStatus.sessionId, 'completed')
  }
  testStore.stopPolling()

  // 2. 调用后端断开连接
  const gunNumber = deviceStore.deviceInfo.gunNumber || deviceStore.selectedSession?.gunNumber || ''
  if (gunNumber) {
    deviceStore.disconnect(gunNumber).catch(() => {})
  }

  // 3. 刷新会话列表
  deviceStore.fetchSessions()
}

function handleStartTest(data: Record<string, unknown>) {
  testStore.startTestWithConfig(data)
}

function handleViewDetail(row: any) {
  console.log('[handleViewDetail] row:', JSON.stringify(row))

  const sid = row.sessionId || ''
  if (!sid || sid === 'undefined' || sid === 'null') {
    ElMessage.warning('该测试记录缺少会话ID，无法查看详情')
    return
  }
  selectedTestId.value = row.id || 0
  selectedSessionId.value = sid
  showDetailModal.value = true
}

function handleViewMessages(id: number) {
  showDetailModal.value = false
  showMessageModal.value = true
}

function handleExportReport() {
  testStore.exportReport()
}
</script>

<style scoped>
.protocol-test-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  height: 100%;
  overflow-y: auto;
}

/* 隐藏但保留布局空间的wrapper */
.section-wrapper {
  transition: opacity 0.2s ease;
}

.section-hidden {
  /* 隐藏内容但保留高度，防止页面跳动 */
  min-height: 300px;
  visibility: hidden;
  pointer-events: none;
  margin: -16px 0; /* 抵消gap，让视觉上更自然 */
}
</style>
