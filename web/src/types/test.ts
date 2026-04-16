export interface PriceRow {
  id: string
  periodType: 'peak' | 'high' | 'flat' | 'valley' | ''
  startTime: string
  endTime: string
  electricityFee: string
  serviceFee: string
}

export interface TestConfig {
  testCase: 'basic_charging' | 'sftp_upgrade' | 'config_download' | ''
  gunNumber: string
  vinCode?: string
  balance?: string
  displayMode?: '0' | '1' | ''
  maxCharge?: string
  duration?: string
  amount?: string
  soc?: string
  stopCode?: string
  priceRows?: PriceRow[]
  firmwareVersion?: string
  configItems?: ConfigItem[]
}

export interface ConfigItem {
  funcCode: number
  payload: Record<string, unknown>
}

export interface TestResult {
  id: number
  sessionId: string
  postNo: number
  protocolName: string
  startTime: string
  endTime: string
  durationMs: number
  totalMessages: number
  successTotal: number
  failTotal: number
  successRate: number
  isPass: boolean
  status: string
}

export interface MessageDetail {
  hex: string
  json: string
}

export interface FuncCodeStat {
  funcCode: string
  direction: string
  totalMessages: number
  successCount: number
  decodeFail: number
  invalidField: number
  messageLoss: number
  successRate: number
}

export interface TestDetail {
  sessionId: string
  startTime: string
  endTime: string
  status: string
  statistics: FuncCodeStat[]
}

export interface TestStatus {
  sessionId: string
  status: string
  progress: number
  stepName: string
  testCase: string
}
