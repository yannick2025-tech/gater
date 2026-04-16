<template>
  <div class="protocol-test-page">
    <!-- 第1块：设备连接/断开 -->
    <DeviceInfoBar
      :gun-number="deviceStore.deviceInfo.gunNumber"
      :is-online="deviceStore.deviceInfo.isOnline"
      :protocol-name="deviceStore.deviceInfo.protocolName"
      :protocol-version="deviceStore.deviceInfo.protocolVersion"
      @connect="handleConnect"
      @disconnect="handleDisconnect"
      @query="handleQuery"
    />

    <!-- 第2块：测试配置 -->
    <TestConfig
      :is-online="deviceStore.deviceInfo.isOnline"
      @start="handleStartTest"
    />

    <!-- 第3块：测试结果 -->
    <TestResults
      :results="testStore.testResults"
      @view-detail="handleViewDetail"
      @export="handleExportReport"
    />

    <!-- Detail Modal -->
    <TestDetailModal
      v-model="showDetailModal"
      :test-id="selectedTestId"
      @view-messages="handleViewMessages"
    />

    <!-- Message View Modal -->
    <MessageViewModal v-model="showMessageModal" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useDeviceStore } from '@/stores/device'
import { useTestStore } from '@/stores/test'
import DeviceInfoBar from '@/components/DeviceInfoBar.vue'
import TestConfig from '@/components/TestConfig.vue'
import TestResults from '@/components/TestResults.vue'
import TestDetailModal from '@/components/TestDetailModal.vue'
import MessageViewModal from '@/components/MessageViewModal.vue'

const deviceStore = useDeviceStore()
const testStore = useTestStore()

const showDetailModal = ref(false)
const showMessageModal = ref(false)
const selectedTestId = ref<number | null>(null)

function handleConnect(gunNumber: string) {
  deviceStore.connect(gunNumber)
}

function handleDisconnect() {
  if (deviceStore.deviceInfo.gunNumber) {
    deviceStore.disconnect(deviceStore.deviceInfo.gunNumber)
  }
}

function handleQuery(gunNumber: string) {
  deviceStore.query(gunNumber)
}

function handleStartTest(data: Record<string, unknown>) {
  testStore.startTestWithConfig(data)
}

function handleViewDetail(id: number) {
  selectedTestId.value = id
  showDetailModal.value = true
}

function handleViewMessages(id: number) {
  showDetailModal.value = false
  showMessageModal.value = true
}

function handleExportReport() {
  testStore.exportReport()
}
</script>

<style scoped>
.protocol-test-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  height: 100%;
  overflow-y: auto;
}
</style>
