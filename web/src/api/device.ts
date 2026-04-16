import request from './request'
import type { DeviceInfo } from '@/types/device'

export function getDeviceStatus(gunNumber: string) {
  return request.get<any, DeviceInfo>('/device/status', { params: { gunNumber } })
}

export function connectDevice(gunNumber: string) {
  return request.post<any, { isOnline: boolean; sessionId: string }>('/device/connect', { gunNumber, action: 'connect' })
}

export function disconnectDevice(gunNumber: string) {
  return request.post<any, { isOnline: boolean }>('/device/disconnect', { gunNumber })
}
