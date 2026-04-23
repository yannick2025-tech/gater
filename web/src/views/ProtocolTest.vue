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

    <!-- 第2块：测试配置（选中会话后始终显示；历史会话时置灰只读） -->
    <div class="section-wrapper" v-if="selectionState !== 'none'">
      <TestConfig
        :can-start-test="hasActiveSession"
        :is-historical="selectionState === 'historical'"
        :is-charging="isCharging"
        :is-charging-stopped="isChargingStopped"
        @start="handleStartTest"
        @stop="handleStopTest"
      />
    </div>

    <!-- 充电信息面板（充电中或充电结束后显示） -->
    <div class="section-wrapper" v-if="chargingInfo">
      <ChargingInfoPanel :info="chargingInfo" />
    </div>

    <!-- 第3块：测试结果（选中会话后显示） -->
    <div class="section-wrapper" v-if="selectionState !== 'none'">
      <TestResults
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
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useDeviceStore } from '@/stores/device'
import { useTestStore } from '@/stores/test'
import type { SessionItem } from '@/types/device'
import DeviceInfoBar from '@/components/DeviceInfoBar.vue'
import TestConfig from '@/components/TestConfig.vue'
import TestResults from '@/components/TestResults.vue'
import TestDetailModal from '@/components/TestDetailModal.vue'
import MessageViewModal from '@/components/MessageViewModal.vue'
import ChargingInfoPanel from '@/components/ChargingInfoPanel.vue'
import { stopTest as stopTestApi, getChargingInfo } from '@/api/test'

const deviceStore = useDeviceStore()
const testStore = useTestStore()

const showDetailModal = ref(false)
const showMessageModal = ref(false)
const selectedTestId = ref<number | null>(null)
const selectedSessionId = ref<string>('')

// 本地测试运行状态（立即响应，不依赖session列表轮询）
const isTestRunning = ref(false)

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

/** 是否有活跃会话且未在测试中（用于控制"开始测试"按钮） */
const hasActiveSession = computed(() => {
  if (selectionState.value !== 'active') return false
  // 本地标记或session列表都标记running时，禁用按钮
  if (isTestRunning.value) return false
  const sess = deviceStore.selectedSession
  if (sess?.testStatus === 'running') return false
  return true
})

// ========== 充电状态管理 ==========
const chargingInfo = ref<any>(null)
const chargingPollTimer = ref<ReturnType<typeof setInterval> | null>(null)

const isCharging = computed(() => {
  // 本地标记测试运行中即可显示充电面板和结束按钮
  return isTestRunning.value
})

const isChargingStopped = computed(() => {
  return chargingInfo.value?.chargingInfo?.isChargingStopped === true
})

function startChargingPolling(sessionId: string) {
  stopChargingPolling()
  // 立即查一次
  fetchChargingInfo(sessionId)
  // 每5秒轮询
  chargingPollTimer.value = setInterval(() => {
    fetchChargingInfo(sessionId)
  }, 5000)
}

function stopChargingPolling() {
  if (chargingPollTimer.value) {
    clearInterval(chargingPollTimer.value)
    chargingPollTimer.value = null
  }
}

async function fetchChargingInfo(sessionId: string) {
  try {
    const data = await getChargingInfo(sessionId)
    chargingInfo.value = data
    // 充电已停止（0x05收到后），标记测试不再运行
    if (data?.chargingInfo?.isChargingStopped) {
      isTestRunning.value = false
    }
  } catch {
    // ignore
  }
}

async function handleStopTest() {
  const sessionId = deviceStore.selectedSession?.sessionId
  if (!sessionId) return
  try {
    await stopTestApi(sessionId)
    ElMessage.success('已发送结束充电指令')
  } catch (e: any) {
    ElMessage.error(e?.message || '结束充电失败')
  }
}

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
  stopChargingPolling()
})

// ========== 事件处理 ==========

function handleSelectSession(session: SessionItem) {
  deviceStore.selectSession(session)
  // 切换会话时重置本地测试状态
  isTestRunning.value = session.testStatus === 'running'
  chargingInfo.value = null
  stopChargingPolling()
  // 选中历史会话时，加载该会话的测试结果
  if (!session.isOnline) {
    testStore.fetchResultsBySession(session.sessionId)
  }
  // 如果选中在线且正在测试的会话，启动充电轮询
  if (session.isOnline && session.testStatus === 'running') {
    startChargingPolling(session.sessionId)
  }
}

function handleBackToList() {
  deviceStore.reset()
  isTestRunning.value = false
  chargingInfo.value = null
  stopChargingPolling()
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
  isTestRunning.value = true
  testStore.startTestWithConfig(data).then(() => {
    // 开始测试后启动充电信息轮询
    const sessionId = deviceStore.selectedSession?.sessionId
    if (sessionId) {
      startChargingPolling(sessionId)
    }
  }).catch(() => {
    isTestRunning.value = false
  })
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
