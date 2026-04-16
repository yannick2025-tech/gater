<template>
  <div class="bg-white p-6 rounded-lg shadow">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-base font-semibold">测试结果</h3>
      <el-button size="small" @click="$emit('page-change', 1)">
        <RefreshCw class="w-4 h-4 mr-1" /> 刷新
      </el-button>
    </div>

    <el-table :data="results" stripe v-loading="loading" empty-text="暂无测试结果">
      <el-table-column prop="sessionId" label="Session ID" width="160" />
      <el-table-column prop="postNo" label="枪编号" width="120" />
      <el-table-column prop="protocolName" label="协议" width="120" />
      <el-table-column label="开始时间" width="170">
        <template #default="{ row }">{{ formatTime(row.startTime) }}</template>
      </el-table-column>
      <el-table-column label="持续时间" width="100">
        <template #default="{ row }">{{ formatDuration(row.durationMs) }}</template>
      </el-table-column>
      <el-table-column label="消息数" width="90" align="right">
        <template #default="{ row }">{{ row.totalMessages }}</template>
      </el-table-column>
      <el-table-column label="成功率" width="90" align="right">
        <template #default="{ row }">
          <span :style="{ color: row.successRate >= 100 ? '#52C41A' : '#EA933F' }">
            {{ row.successRate.toFixed(1) }}%
          </span>
        </template>
      </el-table-column>
      <el-table-column label="结果" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.isPass ? 'success' : 'danger'" size="small">
            {{ row.isPass ? '通过' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" text size="small" @click="$emit('view-detail', row.sessionId)">
            详情
          </el-button>
          <el-button type="primary" text size="small" @click="$emit('export-report', row.sessionId)">
            导出PDF
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="flex justify-end mt-4">
      <el-pagination
        v-model:current-page="currentPage_"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="(page: number) => $emit('page-change', page)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import dayjs from 'dayjs'
import type { TestResult } from '@/types/test'

const props = defineProps<{
  results: TestResult[]
  total: number
  currentPage: number
  pageSize: number
  loading: boolean
}>()

defineEmits<{
  'page-change': [page: number]
  'view-detail': [sessionId: string]
  'export-report': [sessionId: string]
}>()

const currentPage_ = computed(() => props.currentPage)

function formatTime(t: string) {
  return t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'
}

function formatDuration(ms: number) {
  if (!ms) return '-'
  if (ms < 60000) return `${(ms / 1000).toFixed(0)}s`
  return `${(ms / 60000).toFixed(1)}min`
}
</script>
