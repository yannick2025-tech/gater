<template>
  <div class="flex-1 p-6 bg-gray-50 overflow-auto">
    <DeviceInfoBar
      :gun-number="deviceStore.deviceInfo.gunNumber"
      :protocol-name="deviceStore.deviceInfo.protocolName"
      :protocol-version="deviceStore.deviceInfo.protocolVersion"
      :is-online="deviceStore.deviceInfo.isOnline"
      :loading="deviceStore.loading"
      @disconnect="handleDisconnect"
      @query="handleQuery"
    />

    <TestConfig
      :gun-number="deviceStore.deviceInfo.gunNumber"
      :is-online="deviceStore.deviceInfo.isOnline"
      :test-loading="testStore.loading"
      :current-status="testStore.currentStatus"
      @start-test="handleStartTest"
      @start-config="handleStartConfig"
    />

    <TestResults
      :results="testStore.testResults"
      :total="testStore.total"
      :current-page="testStore.currentPage"
      :page-size="testStore.pageSize"
      :loading="testStore.loading"
      @page-change="testStore.fetchResults"
      @view-detail="handleViewDetail"
      @export-report="handleExportReport"
    />

    <TestDetailModal
      v-model:visible="detailVisible"
      :session-id="selectedSessionId"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DeviceInfoBar from '@/components/DeviceInfoBar.vue'
import TestConfig from '@/components/TestConfig.vue'
import TestResults from '@/components/TestResults.vue'
import TestDetailModal from '@/components/TestDetailModal.vue'
import { useDeviceStore } from '@/stores/device'
import { useTestStore } from '@/stores/test'
import { exportReport } from '@/api/test'
import { ElMessage } from 'element-plus'
import type { ConfigItem } from '@/types/test'

const deviceStore = useDeviceStore()
const testStore = useTestStore()

const detailVisible = ref(false)
const selectedSessionId = ref('')

onMounted(() => {
  testStore.fetchResults(1)
})

function handleQuery(gunNumber: string) {
  deviceStore.fetchStatus(gunNumber)
}

function handleDisconnect() {
  if (deviceStore.deviceInfo.gunNumber) {
    deviceStore.disconnect(deviceStore.deviceInfo.gunNumber)
  }
}

function handleStartTest(testCase: string, gunNumber: string) {
  testStore.startTest(testCase, gunNumber)
}

function handleStartConfig(gunNumber: string, items: ConfigItem[]) {
  testStore.startConfigTest(gunNumber, items)
}

function handleViewDetail(sessionId: string) {
  selectedSessionId.value = sessionId
  detailVisible.value = true
}

async function handleExportReport(sessionId: string) {
  try {
    const data = await exportReport(sessionId)
    if (data.pdfUrl) {
      window.open(data.pdfUrl, '_blank')
    }
    ElMessage.success('报告生成成功')
  } catch {
    ElMessage.error('报告生成失败')
  }
}
</script>
