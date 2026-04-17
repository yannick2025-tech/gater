import request from './request'
import type { TestResult, TestDetail, TestStatus, ConfigItem, MessageArchive } from '@/types/test'

// 开始测试（基于已存在的TCP会话）
export function startTest(testCase: string, sessionId: string = '', params?: Record<string, unknown>) {
  return request.post<any, TestStatus>('/test/start', { testCase, sessionId, params })
}

export function getTestStatus(sessionId: string) {
  return request.get<any, TestStatus>(`/test/status/${sessionId}`)
}

export function getTestResults(page: number, pageSize: number, startTime?: string, endTime?: string, sessionId?: string) {
  return request.get<any, { total: number; page: number; pageSize: number; list: TestResult[] }>('/test/results', {
    params: { page, pageSize, startTime, endTime, sessionId },
  })
}

export function getTestDetail(sessionId: string) {
  return request.get<any, TestDetail>(`/test/detail/${sessionId}`)
}

export function getMessageArchives(sessionId: string, funcCode: string, status: string) {
  return request.get<any, MessageArchive[]>(`/test/messages/${sessionId}`, {
    params: { sessionID: sessionId, funcCode, status },
  })
}

export function decodeMessage(hex: string) {
  return request.post<any, { json: string }>('/test/decode', { hex })
}

export function exportReport(sessionId: string) {
  return request.post<any, { sessionId: string; pdfUrl: string; pdfPath: string }>('/test/export', { sessionId })
}

export function configDownload(gunNumber: string, items: ConfigItem[]) {
  return request.post<any, TestStatus>('/test/config', { gunNumber, items })
}
