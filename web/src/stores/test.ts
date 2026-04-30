import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { TestStatus } from '@/types/test'
import { getTestResults, getTestStatus, startTest as startTestApi, configDownload, exportReport as exportReportApi } from '@/api/test'
import type { ConfigItem } from '@/types/test'
import { ElMessage } from 'element-plus'
import { useDeviceStore } from '@/stores/device'
import { useRouter } from 'vue-router'

export const useTestStore = defineStore('test', () => {
  const testResults = ref<any[]>([])
  const total = ref(0)
  const currentPage = ref(1)
  const pageSize = ref(10)
  const loading = ref(false)
  const currentStatus = ref<TestStatus | null>(null)
  const pollTimer = ref<ReturnType<typeof setInterval> | null>(null)
  const currentScenarioId = ref<string>('')  // 当前正在运行的测试场景ID

  async function fetchResults(page = 1) {
    loading.value = true
    try {
      const data = await getTestResults(page, pageSize.value)
      testResults.value = data.list || []
      total.value = data.total || 0
      currentPage.value = data.page || page
    } finally {
      loading.value = false
    }
  }

  async function startTest(testCase: string, gunNumber: string, params?: Record<string, unknown>) {
    loading.value = true
    try {
      const data = await startTestApi(testCase, gunNumber, params)
      currentStatus.value = data
      currentScenarioId.value = data.scenarioId || ''

      // 立即插入一条运行中记录到结果列表（让用户看到测试已开始）
      const now = new Date().toISOString().slice(0, 19).replace('T', ' ')
      const runningRecord: Record<string, any> = {
        id: Date.now(),
        sessionId: data.sessionId,
        scenarioId: data.scenarioId || '',
        protocolName: testCase,
        startTime: now,
        endTime: '',
        durationMs: 0,
        totalMessages: 0,
        successTotal: 0,
        failTotal: 0,
        successRate: 0,
        isPass: false,
        status: 'running',
      }
      testResults.value.unshift(runningRecord)
      total.value += 1

      startPolling(data.sessionId)
      return data
    } finally {
      loading.value = false
    }
  }

  function startTestWithConfig(data: Record<string, unknown>) {
    const scenario = (data.scenario as string) || 'basic_charging'
    const deviceStore = useDeviceStore()
    // 使用选中的sessionId（来自会话列表）
    const sessionId = deviceStore.deviceInfo.sessionId || deviceStore.selectedSession?.sessionId || ''
    if (!sessionId) {
      ElMessage.warning('请先选择一个会话')
      return Promise.reject(new Error('sessionId missing'))
    }
    return startTestBySession(sessionId, scenario, data)
  }

  async function startTestBySession(sessionId: string, testCase: string, params?: Record<string, unknown>) {
    loading.value = true
    try {
      const data = await startTestApi(testCase, sessionId, params)
      currentStatus.value = data
      currentScenarioId.value = data.scenarioId || ''

      // 立即插入一条运行中记录到结果列表
      const now = new Date().toISOString().slice(0, 19).replace('T', ' ')
      const runningRecord: Record<string, any> = {
        id: Date.now(),
        sessionId: sessionId,
        scenarioId: data.scenarioId || '',
        protocolName: testCase,
        startTime: now,
        endTime: '',
        durationMs: 0,
        totalMessages: 0,
        successTotal: 0,
        failTotal: 0,
        successRate: 0,
        isPass: false,
        status: 'running',
      }
      testResults.value.unshift(runningRecord)
      total.value += 1

      startPolling(sessionId)
      return data
    } finally {
      loading.value = false
    }
  }

  // 当前活跃的 sessionId（供外部设置，解决断开连接后 selectedSession 被清空的问题）
  const currentExportSessionId = ref('')

  function setExportSessionId(sid: string) {
    currentExportSessionId.value = sid
  }

  /**
   * 按优先级获取当前会话 ID：
   * 1. 显式设置的 currentExportSessionId（组件主动设置，最可靠）
   * 2. deviceStore.selectedSession（正常选中状态）
   * 3. URL 路由参数（新开浏览器/刷新后从 URL 恢复）
   */
  function resolveSessionId(): string {
    // 优先级1：显式设置
    if (currentExportSessionId.value) return currentExportSessionId.value
    // 优先级2：选中的会话
    const deviceStore = useDeviceStore()
    if (deviceStore.selectedSession?.sessionId) return deviceStore.selectedSession.sessionId
    // 优先级3：URL 路由参数
    const router = useRouter()
    const urlSid = router.currentRoute.value.params.sessionId as string
    if (urlSid) return urlSid
    return ''
  }

  function exportReport() {
    const sessionId = resolveSessionId()
    if (!sessionId) {
      ElMessage.warning('请先选择一个会话')
      return
    }
    ElMessage.info('正在生成测试报告...')
    exportReportApi(sessionId).then((res) => {
      const zipUrl = res.zipUrl
      if (zipUrl) // 通过 <a> 标签触发浏览器下载 ZIP 文件
      {
        const link = document.createElement('a')
        link.href = zipUrl
        link.download = '' // 服务端返回 Content-Disposition，此处留空即可
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        ElMessage.success('测试报告已生成')
      } else {
        ElMessage.error('生成报告失败：未返回报告路径')
      }
    }).catch((e: any) => {
      ElMessage.error(e?.message || '导出测试报告失败')
    })
  }

  async function startConfigTest(gunNumber: string, items: ConfigItem[]) {
    loading.value = true
    try {
      const data = await configDownload(gunNumber, items)
      currentStatus.value = data
      startPolling(data.sessionId)
      return data
    } finally {
      loading.value = false
    }
  }

  function startPolling(sessionId: string) {
    stopPolling()
    pollTimer.value = setInterval(async () => {
      try {
        const data = await getTestStatus(sessionId)
        currentStatus.value = data
        if (data.status !== 'running') {
          stopPolling()
          markTestCompleted(sessionId, data.status === 'completed' ? 'completed' : 'completed', currentScenarioId.value || undefined)
          fetchResults(1)
        }
      } catch {
        // session不存在或已结束，标记为完成
        stopPolling()
        markTestCompleted(sessionId, 'completed', currentScenarioId.value || undefined)
        fetchResults(1)
      }
    }, 3000)
  }

  function stopPolling() {
    if (pollTimer.value) {
      clearInterval(pollTimer.value)
      pollTimer.value = null
    }
  }

  // 标记测试记录为已完成（断开连接/停止测试时调用）
  // 优先按 scenarioId 精确匹配（同一 session 可能有多个场景），回退到 sessionId
  function markTestCompleted(sessionId: string, finalStatus: string, scenarioId?: string) {
    const record = testResults.value.find(r => {
      if (scenarioId) return r.scenarioId === scenarioId && r.status === 'running'
      return r.sessionId === sessionId && r.status === 'running'
    })
    if (record) {
      const now = new Date().toISOString().slice(0, 19).replace('T', ' ')
      record.status = finalStatus
      record.endTime = now
      record.isPass = finalStatus === 'completed'
      if (record.startTime) {
        record.durationMs = new Date(now).getTime() - new Date(record.startTime).getTime()
      }
    }
  }

  /** 按sessionId加载该会话的所有测试场景结果（用于历史/断开后查看报告） */
  async function fetchResultsBySession(sessionId: string) {
    loading.value = true
    try {
      const data = await getTestResults(1, 100, '', '', sessionId)
      testResults.value = data.list || []
      total.value = data.total || 0
      currentPage.value = 1
    } finally {
      loading.value = false
    }
  }

  // 页面加载时自动拉取历史测试结果（从MySQL）
  fetchResults(1)

  return {
    testResults, total, currentPage, pageSize, loading, currentStatus, currentScenarioId,
    fetchResults, fetchResultsBySession, startTest, startTestWithConfig, startConfigTest, exportReport,
    startPolling, stopPolling, markTestCompleted,
    setExportSessionId, currentExportSessionId,
  }
})
