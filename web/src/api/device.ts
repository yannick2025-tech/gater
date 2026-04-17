import request from './request'
import type { DeviceInfo, SessionItem } from '@/types/device'

// 获取所有活跃的TCP会话列表
export function getSessions() {
  return request.get<any, { total: number; list: SessionItem[] }>('/sessions')
}

export function getDeviceStatus(gunNumber: string) {
  return request.get<any, DeviceInfo>('/device/status', { params: { gunNumber } })
}

export function connectDevice(gunNumber: string) {
  return request.post<any, { isOnline: boolean; sessionId: string }>('/device/connect', { gunNumber, action: 'connect' })
}

export function disconnectDevice(gunNumber: string) {
  return request.post<any, { isOnline: boolean }>('/device/disconnect', { gunNumber })
}
