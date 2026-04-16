import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { TestStatus } from '@/types/test'
import { getTestResults, getTestStatus, startTest as startTestApi, configDownload } from '@/api/test'
import type { ConfigItem } from '@/types/test'
import { ElMessage } from 'element-plus'
import { useDeviceStore } from '@/stores/device'

export const useTestStore = defineStore('test', () => {
  const testResults = ref<any[]>([])
  const total = ref(0)
  const currentPage = ref(1)
  const pageSize = ref(10)
  const loading = ref(false)
  const currentStatus = ref<TestStatus | null>(null)
  const pollTimer = ref<ReturnType<typeof setInterval> | null>(null)

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

      // 立即插入一条运行中记录到结果列表（让用户看到测试已开始）
      const now = new Date().toISOString().slice(0, 19).replace('T', ' ')
      const runningRecord: Record<string, any> = {
        id: Date.now(),
        sessionId: data.sessionId,
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
    const gunNumber = deviceStore.deviceInfo.gunNumber
    if (!gunNumber) {
      ElMessage.warning('请先连接设备')
      return Promise.reject(new Error('gunNumber missing'))
    }
    return startTest(scenario, gunNumber, data)
  }

  function exportReport() {
    ElMessage.info('导出功能开发中...')
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
          fetchResults(1)
        }
      } catch {
        stopPolling()
      }
    }, 3000)
  }

  function stopPolling() {
    if (pollTimer.value) {
      clearInterval(pollTimer.value)
      pollTimer.value = null
    }
  }

  // 页面加载时自动拉取历史测试结果（从MySQL）
  fetchResults(1)

  return {
    testResults, total, currentPage, pageSize, loading, currentStatus,
    fetchResults, startTest, startTestWithConfig, startConfigTest, exportReport,
    startPolling, stopPolling,
  }
})
