import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DeviceInfo } from '@/types/device'
import { getDeviceStatus, disconnectDevice } from '@/api/device'

export const useDeviceStore = defineStore('device', () => {
  const deviceInfo = ref<DeviceInfo>({
    gunNumber: '',
    protocolName: '',
    protocolVersion: '',
    isOnline: false,
  })

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
      // 调用后端接口查询设备状态（连接成功即认为在线）
      const data = await getDeviceStatus(gunNumber)
      deviceInfo.value = { ...deviceInfo.value, ...data }
      // 连接成功强制设为在线
      deviceInfo.value.isOnline = true
      // 确保协议信息也同步过来
      if (!deviceInfo.value.protocolName) {
        deviceInfo.value.protocolName = data.protocolName || ''
      }
      if (!deviceInfo.value.protocolVersion) {
        deviceInfo.value.protocolVersion = data.protocolVersion || ''
      }
    } catch (e) {
      // 接口调用失败保持离线
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
  }

  return { deviceInfo, loading, fetchStatus, query, connect, disconnect, reset }
})
