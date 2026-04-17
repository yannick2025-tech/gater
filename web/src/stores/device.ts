import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DeviceInfo, SessionItem } from '@/types/device'
import { getDeviceStatus, disconnectDevice, connectDevice, getSessions } from '@/api/device'

export const useDeviceStore = defineStore('device', () => {
  const deviceInfo = ref<DeviceInfo>({
    gunNumber: '',
    protocolName: '',
    protocolVersion: '',
    isOnline: false,
  })

  // 当前选中的session（从会话列表中选择）
  const selectedSession = ref<SessionItem | null>(null)
  const sessionList = ref<SessionItem[]>([])

  const loading = ref(false)

  async function fetchStatus(gunNumber: string) {
    loading.value = true
    try {
      const data = await getDeviceStatus(gunNumber)
      deviceInfo.value = { ...deviceInfo.value, ...data }
    } finally {
      loading.value = false
    }
  }

  function query(gunNumber: string) {
    deviceInfo.value.gunNumber = gunNumber
    return fetchStatus(gunNumber)
  }

  async function connect(gunNumber: string) {
    loading.value = true
    try {
      deviceInfo.value.gunNumber = gunNumber
      await connectDevice(gunNumber)
      deviceInfo.value.isOnline = true
      if (!deviceInfo.value.protocolName) {
        deviceInfo.value.protocolName = 'XX标准协议'
      }
      if (!deviceInfo.value.protocolVersion) {
        deviceInfo.value.protocolVersion = 'v1.6.0'
      }
    } catch (e) {
      deviceInfo.value.isOnline = false
      throw e
    } finally {
      loading.value = false
    }
  }

  async function disconnect(gunNumber: string) {
    loading.value = true
    try {
      await disconnectDevice(gunNumber)
    } finally {
      reset()
    }
  }

  function reset() {
    deviceInfo.value = { gunNumber: '', protocolName: '', protocolVersion: '', isOnline: false }
    selectedSession.value = null
  }

  // 获取所有活跃的TCP会话列表（充电桩主动连接gater产生的）
  async function fetchSessions() {
    try {
      const data = await getSessions()
      sessionList.value = data.list || []
    } catch (e) {
      sessionList.value = []
    }
  }

  // 选择一个会话进行测试
  function selectSession(session: SessionItem | null) {
    selectedSession.value = session
    if (session) {
      deviceInfo.value.gunNumber = session.gunNumber
      deviceInfo.value.sessionId = session.sessionId
      deviceInfo.value.isOnline = session.isOnline   // 保持原始在线状态，不硬编码true
      deviceInfo.value.protocolName = session.protocolName || 'XX标准协议'
      deviceInfo.value.protocolVersion = session.protocolVersion || 'v1.6.0'
    }
  }

  // 定期刷新会话列表
  let sessionTimer: ReturnType<typeof setInterval> | null = null
  function startSessionPolling(intervalMs = 10000) {
    stopSessionPolling()
    fetchSessions() // 立即拉取一次
    sessionTimer = setInterval(fetchSessions, intervalMs)
  }
  function stopSessionPolling() {
    if (sessionTimer) {
      clearInterval(sessionTimer)
      sessionTimer = null
    }
  }

  return {
    deviceInfo, loading, fetchStatus, query, connect, disconnect, reset,
    selectedSession, sessionList, fetchSessions, selectSession,
    startSessionPolling, stopSessionPolling,
  }
})
