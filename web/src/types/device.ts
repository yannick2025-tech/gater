export interface DeviceInfo {
  gunNumber: string
  protocolName: string
  protocolVersion: string
  isOnline: boolean
  sessionId?: string
  authState?: string
  lastHeartbeat?: string
}
