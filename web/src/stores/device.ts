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

  async function disconnect(gunNumber: string) {
    loading.value = true
    try {
      const data = await disconnectDevice(gunNumber)
      deviceInfo.value.isOnline = data.isOnline
    } finally {
      loading.value = false
    }
  }

  function reset() {
    deviceInfo.value = { gunNumber: '', protocolName: '', protocolVersion: '', isOnline: false }
  }

  return { deviceInfo, loading, fetchStatus, disconnect, reset }
})
