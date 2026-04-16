<template>
  <el-dialog
    v-model="visible"
    title="测试详情"
    width="720px"
    :close-on-click-modal="false"
    class="detail-dialog"
  >
    <div class="detail-body">
      <!-- Left: Stats -->
      <div class="stats-panel">
        <h4 class="panel-title">测试统计</h4>
        <div class="stat-grid">
          <div class="stat-item">
            <span class="stat-value">{{ detail.totalMessages }}</span>
            <span class="stat-label">总报文数</span>
          </div>
          <div class="stat-item">
            <span class="stat-value success">{{ detail.successCount }}</span>
            <span class="stat-label">通过</span>
          </div>
          <div class="stat-item">
            <span class="stat-value danger">{{ detail.failureCount }}</span>
            <span class="stat-label">失败</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ detail.duration }}</span>
            <span class="stat-label">耗时</span>
          </div>
        </div>
        <div class="stat-rate">
          <span>成功率：</span>
          <strong>{{ detail.successRate }}%</strong>
        </div>
      </div>

      <!-- Right: Summary Table -->
      <div class="summary-panel">
        <h4 class="panel-title">结果汇总</h4>
        <el-table
          :data="detail.summary"
          size="small"
          border
          style="width: 100%"
          max-height="320px"
          :header-cell-style="{ background: '#fafafa', color: '#666', fontWeight: '500', fontSize: '12px' }"
        >
          <el-table-column prop="step" label="步骤" min-width="80" />
          <el-table-column prop="messageType" label="报文类型" min-width="100" />
          <el-table-column prop="result" label="结果" width="70" align="center">
            <template #default="{ row }">
              <el-tag
                :type="row.result === 'pass' ? 'success' : 'danger'"
                size="small"
                effect="light"
              >
                {{ row.result === 'pass' ? '通过' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
        </el-table>
      </div>
    </div>

    <!-- Footer -->
    <template #footer>
      <el-button
        v-if="testId"
        @click="$emit('view-messages', testId)"
        type="primary"
        plain
        size="small"
      >
        查看测试结果中的报文
      </el-button>
      <el-button @click="visible = false" size="small">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  modelValue: boolean
  testId?: number | null
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  'view-messages': [id: number]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

// Mock data - replace with real API response
const detail = computed(() => ({
  totalMessages: 24,
  successCount: 22,
  failureCount: 2,
  duration: '3m 12s',
  successRate: 91.7,
  summary: [
    { step: '1', messageType: 'BootNotification', result: 'pass', remark: '设备启动通知' },
    { step: '2', messageType: 'Heartbeat', result: 'pass', remark: '心跳正常' },
    { step: '3', messageType: 'StatusNotification', result: 'pass', remark: '状态上报' },
    { step: '4', messageType: 'Authorize', result: 'fail', remark: '鉴权超时' },
    { step: '5', messageType: 'StartTransaction', result: 'pass', remark: '开始充电' },
    { step: '6', messageType: 'MeterValues', result: 'pass', remark: '计量数据' },
    { step: '7', messageType: 'StopTransaction', result: 'fail', remark: '停止异常' },
  ],
}))
</script>

<style scoped>
.detail-body {
  display: flex;
  gap: 20px;
}

.stats-panel {
  flex: 0 0 200px;
  padding: 16px;
  background-color: #f8f9fb;
  border-radius: 8px;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  margin: 0 0 16px;
}

.stat-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 16px;
}

.stat-item {
  text-align: center;
  padding: 10px 4px;
  background: #fff;
  border-radius: 6px;
}

.stat-value {
  display: block;
  font-size: 20px;
  font-weight: 700;
  color: #333;
  line-height: 1.3;
}

.stat-value.success {
  color: #67c23a;
}

.stat-value.danger {
  color: #f56c6c;
}

.stat-label {
  font-size: 11px;
  color: #999;
  margin-top: 2px;
}

.stat-rate {
  font-size: 13px;
  color: #666;
  text-align: center;
  padding-top: 8px;
  border-top: 1px solid #eee;
}

.stat-rate strong {
  color: #409eff;
  font-size: 18px;
}

.summary-panel {
  flex: 1;
  min-width: 0;
}
</style>
