import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { TestResult, TestStatus } from '@/types/test'
import { getTestResults, getTestStatus, startTest as startTestApi, configDownload } from '@/api/test'
import type { ConfigItem } from '@/types/test'

export const useTestStore = defineStore('test', () => {
  const testResults = ref<TestResult[]>([])
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
      startPolling(data.sessionId)
      return data
    } finally {
      loading.value = false
    }
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

  return {
    testResults, total, currentPage, pageSize, loading, currentStatus,
    fetchResults, startTest, startConfigTest, startPolling, stopPolling,
  }
})
