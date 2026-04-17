<template>
  <div class="protocol-test-page">
    <!-- 第1块：设备连接/断开（会话列表选择器） -->
    <DeviceInfoBar
      :gun-number="deviceStore.deviceInfo.gunNumber"
      :is-online="deviceStore.deviceInfo.isOnline"
      :protocol-name="deviceStore.deviceInfo.protocolName"
      :protocol-version="deviceStore.deviceInfo.protocolVersion"
      :sessions="deviceStore.sessionList"
      @select-session="handleSelectSession"
      @disconnect="handleDisconnect"
      @refresh-sessions="deviceStore.fetchSessions"
    />

    <!-- 第2块：测试配置 -->
    <TestConfig
      :is-online="deviceStore.deviceInfo.isOnline"
      @start="handleStartTest"
    />

    <!-- 第3块：测试结果 -->
    <TestResults
      :results="testStore.testResults"
      @view-detail="handleViewDetail"
      @export="handleExportReport"
    />

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
import { ref, onMounted, onUnmounted } from 'vue'
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

// 页面挂载时启动会话列表轮询
onMounted(() => {
  deviceStore.startSessionPolling(10000)
})

// 页面卸载时停止所有轮询
onUnmounted(() => {
  deviceStore.stopSessionPolling()
  testStore.stopPolling()
})

function handleSelectSession(session: SessionItem | null) {
  if (session) {
    deviceStore.selectSession(session)
  } else {
    deviceStore.selectedSession = null
  }
}

function handleDisconnect() {
  // 1. 先停止轮询，标记测试为已完成
  if (testStore.currentStatus?.sessionId) {
    testStore.markTestCompleted(testStore.currentStatus.sessionId, 'completed')
  }
  testStore.stopPolling()

  // 2. 调用后端断开连接（如果当前有选中的session）
  const gunNumber = deviceStore.deviceInfo.gunNumber || deviceStore.selectedSession?.gunNumber || ''
  if (gunNumber) {
    deviceStore.disconnect(gunNumber).catch(() => {})
  }

  // 3. 刷新会话列表
  deviceStore.fetchSessions()
}

function handleQuery(gunNumber: string) {
  deviceStore.query(gunNumber)
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
</style>
