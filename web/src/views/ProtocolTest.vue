<template>
  <div class="protocol-test-page">
    <!-- 第1块：会话选择 / 设备信息（双模式） -->
    <DeviceInfoBar
      :gun-number="disconnectedSession?.gunNumber || deviceStore.deviceInfo.gunNumber"
      :is-online="isDisconnected ? false : deviceStore.deviceInfo.isOnline"
      :protocol-name="deviceStore.deviceInfo.protocolName"
      :protocol-version="deviceStore.deviceInfo.protocolVersion"
      :sessions="deviceStore.sessionList"
      :mode="viewMode"
      @select-session="handleSelectSession"
      @disconnect="handleDisconnect"
      @refresh-sessions="deviceStore.fetchSessions"
      @back-to-list="handleBackToList"
    />

    <!-- 第2块：测试配置（选中活跃会话且未断开时显示；历史/断开后隐藏） -->
    <div class="section-wrapper" v-if="selectionState === 'active'">
      <TestConfig
        :can-start-test="hasActiveSession"
        :is-historical="false"
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

    <!-- 第3块：测试结果（选中会话或断开后显示当前会话的结果） -->
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
import { useRouter, useRoute } from 'vue-router'
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

const router = useRouter()
const route = useRoute()

const deviceStore = useDeviceStore()
const testStore = useTestStore()

const showDetailModal = ref(false)
const showMessageModal = ref(false)
const selectedTestId = ref<number | null>(null)
const selectedSessionId = ref<string>('')

// 本地测试运行状态（立即响应，不依赖session列表轮询）
const isTestRunning = ref(false)
// 断开连接后保持显示当前会话的测试结果（不退回到会话列表）
const isDisconnected = ref(false)
const disconnectedSession = ref<SessionItem | null>(null)

// ========== 三态逻辑：none / active / historical / disconnected ==========

type SelectionState = 'none' | 'active' | 'historical' | 'disconnected'

/** 当前选择状态 */
const selectionState = computed<SelectionState>(() => {
  if (isDisconnected.value && disconnectedSession.value) return 'disconnected'
  const sess = deviceStore.selectedSession
  if (!sess) return 'none'
  if (sess.isOnline) return 'active'
  return 'historical'
})

/** DeviceInfoBar 的显示模式：断开后也显示 detail 模式，不显示会话列表 */
const viewMode = computed<'select' | 'detail'>(() => {
  if (selectionState.value === 'none') return 'select'
  return 'detail'  // active / historical / disconnected 都用 detail 模式
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
 * - 断开后：显示当前断开会话的所有测试场景记录
 * - 其他：始终只显示当前选中会话对应的测试记录
 */
const filteredResults = computed(() => {
  // 断开状态：使用保存的断开会话信息过滤
  if (isDisconnected.value && disconnectedSession.value) {
    return testStore.testResults.filter(r => r.sessionId === disconnectedSession.value!.sessionId)
  }
  const sess = deviceStore.selectedSession
  if (!sess) return []
  return testStore.testResults.filter(r => r.sessionId === sess.sessionId)
})

// 监听session列表变化，同步本地测试状态
watch(() => deviceStore.selectedSession, (sess) => {
  if (sess && sess.testStatus === 'running') {
    isTestRunning.value = true
  } else if (sess && sess.testStatus !== 'running' && isTestRunning.value && !chargingInfo.value?.chargingInfo) {
    // session列表显示测试不再运行且无充电信息，则重置
    isTestRunning.value = false
  }
})

// 页面挂载时启动会话列表轮询，并根据URL参数或运行状态恢复选中会话
onMounted(async () => {
  await deviceStore.fetchSessions()

  // 优先级1：URL中有 sessionId 参数（用户直接访问 /session/:id 或刷新页面）
  const urlSessionId = route.params.sessionId as string
  if (urlSessionId) {
    const matchedSession = deviceStore.sessionList.find(
      (s: SessionItem) => s.sessionId === urlSessionId
    )
    if (matchedSession) {
      handleSelectSession(matchedSession)
      // 如果是已断开的历史会话，标记为 disconnected 状态以保持详情视图
      if (!matchedSession.isOnline) {
        isDisconnected.value = true
        disconnectedSession.value = { ...matchedSession }
      }
    } else {
      // URL中的sessionId在列表中未找到（可能已被清理），回退到列表页
      router.replace('/')
    }
    deviceStore.startSessionPolling(10000)
    return
  }

  // 优先级2：自动选中正在测试的在线会话（首页默认行为）
  const runningSession = deviceStore.sessionList.find(
    (s: SessionItem) => s.isOnline && s.testStatus === 'running'
  )
  if (runningSession) {
    handleSelectSession(runningSession)
  }
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
  // 选择新会话时清除断开状态
  isDisconnected.value = false
  disconnectedSession.value = null
  // 切换会话时重置本地测试状态
  isTestRunning.value = session.testStatus === 'running'
  chargingInfo.value = null
  stopChargingPolling()
  // 选中会话时，加载该会话的测试结果
  testStore.fetchResultsBySession(session.sessionId)
  // 如果选中在线且正在测试的会话，启动充电轮询
  if (session.isOnline && session.testStatus === 'running') {
    startChargingPolling(session.sessionId)
  }
  // ★ 同步URL（支持浏览器前进后退、分享链接、刷新恢复）
  router.push({ name: 'SessionDetail', params: { sessionId: session.sessionId } })
}

function handleBackToList() {
  deviceStore.reset()
  isTestRunning.value = false
  isDisconnected.value = false
  disconnectedSession.value = null
  chargingInfo.value = null
  stopChargingPolling()
  // ★ 回到首页（URL同步为 /）
  router.replace({ name: 'ProtocolTest' })
}

function handleDisconnect() {
  // 1. 保存当前会话信息（断开后用于显示测试结果）
  const sess = deviceStore.selectedSession
  if (sess) {
    disconnectedSession.value = { ...sess }
  }
  isDisconnected.value = true

  // 2. 停止所有轮询
  testStore.stopPolling()
  stopChargingPolling()

  // 3. 标记测试为已完成（优先使用currentStatus，回退到selectedSession）
  const sessionId = testStore.currentStatus?.sessionId || deviceStore.selectedSession?.sessionId
  if (sessionId) {
    testStore.markTestCompleted(sessionId, 'completed', testStore.currentScenarioId || undefined)
  }

  // 4. 重置测试运行状态
  isTestRunning.value = false

  // 5. 调用后端断开连接
  const gunNumber = deviceStore.deviceInfo.gunNumber || deviceStore.selectedSession?.gunNumber || ''
  if (gunNumber) {
    deviceStore.disconnect(gunNumber).catch(() => {})
  }

  // 6. 刷新会话列表（后台更新，不切换到会话列表视图）
  deviceStore.fetchSessions()

  // ★ 7. 确保URL保持为 /session/:id（刷新后仍能恢复到此会话的详情视图）
  if (sessionId && route.params.sessionId !== sessionId) {
    router.replace({ name: 'SessionDetail', params: { sessionId } })
  }
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
