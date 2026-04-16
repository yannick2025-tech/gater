import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DeviceInfo } from '@/types/device'
import { getDeviceStatus, disconnectDevice, connectDevice } from '@/api/device'

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
      // 调用后端连接注册接口，在 connRegistry 中标记设备为已连接
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
  }

  return { deviceInfo, loading, fetchStatus, query, connect, disconnect, reset }
})
