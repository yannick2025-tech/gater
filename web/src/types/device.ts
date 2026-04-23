export interface DeviceInfo {
  gunNumber: string
  protocolName: string
  protocolVersion: string
  isOnline: boolean
  sessionId?: string
  authState?: string
  lastHeartbeat?: string
}

// TCP会话列表项（充电桩主动连接gater时产生）
export interface SessionItem {
  sessionId: string
  postNo: number
  gunNumber: string
  authState: string       // none | pending | authenticated
  isOnline: boolean
  protocolName: string
  protocolVersion: string
  connectedAt: string     // 会话创建时间
  lastActive: string      // 最后活跃时间
  testStatus?: string     // running | completed | failed（如果有运行中的测试）
  testCase?: string       // 测试用例名称
  testProgress?: number   // 测试进度
}
