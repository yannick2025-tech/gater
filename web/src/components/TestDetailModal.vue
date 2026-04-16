<template>
  <el-dialog
    v-model="dialogVisible"
    title="测试用例详情"
    width="90%"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <div v-loading="loading" class="space-y-6">
      <template v-if="detail">
        <div class="grid grid-cols-3 gap-4 p-4 bg-gray-50 rounded">
          <div>
            <div class="text-gray-500 text-xs mb-1">Session ID</div>
            <div class="font-medium text-sm">{{ detail.sessionId }}</div>
          </div>
          <div>
            <div class="text-gray-500 text-xs mb-1">开始时间</div>
            <div class="font-medium text-sm">{{ formatTime(detail.startTime) }}</div>
          </div>
          <div>
            <div class="text-gray-500 text-xs mb-1">结束时间</div>
            <div class="font-medium text-sm">{{ formatTime(detail.endTime) }}</div>
          </div>
        </div>

        <div class="flex items-center justify-center py-2">
          <el-tag
            :type="detail.status === 'pass' ? 'success' : 'danger'"
            size="large"
            :style="{
              backgroundColor: detail.status === 'pass' ? '#52C41A' : '#FF4D4F',
              color: 'white',
              fontSize: '1.1rem',
              padding: '8px 28px',
              border: 'none',
            }"
          >
            {{ detail.status === 'pass' ? '测试通过' : '测试失败' }}
          </el-tag>
        </div>

        <el-table :data="detail.statistics" border stripe size="small">
          <el-table-column prop="funcCode" label="功能码" width="80" />
          <el-table-column prop="direction" label="方向" width="120" />
          <el-table-column prop="totalMessages" label="总数" width="70" align="right" />
          <el-table-column prop="successCount" label="成功" width="70" align="right" />
          <el-table-column label="消息丢失" width="80" align="right">
            <template #default="{ row }">
              <span :class="{ 'text-red-600': row.messageLoss > 0 }">{{ row.messageLoss }}</span>
            </template>
          </el-table-column>
          <el-table-column label="解码失败" width="80" align="right">
            <template #default="{ row }">
              <span :class="{ 'text-red-600': row.decodeFail > 0 }">{{ row.decodeFail }}</span>
            </template>
          </el-table-column>
          <el-table-column label="字段非法" width="80" align="right">
            <template #default="{ row }">
              <span :class="{ 'text-red-600': row.invalidField > 0 }">{{ row.invalidField }}</span>
            </template>
          </el-table-column>
          <el-table-column label="成功率" width="80" align="right">
            <template #default="{ row }">
              <span :style="{ color: row.successRate >= 100 ? '#52C41A' : '#EA933F' }">
                {{ row.successRate.toFixed(1) }}%
              </span>
            </template>
          </el-table-column>
        </el-table>

        <div class="grid grid-cols-4 gap-4 p-4 bg-gray-50 rounded text-center">
          <div>
            <div class="text-gray-500 text-xs mb-1">总消息数</div>
            <div class="font-medium">{{ totalMessages }}</div>
          </div>
          <div>
            <div class="text-gray-500 text-xs mb-1">成功总数</div>
            <div class="font-medium" style="color: #52C41A">{{ successTotal }}</div>
          </div>
          <div>
            <div class="text-gray-500 text-xs mb-1">失败总数</div>
            <div class="font-medium" style="color: #FF4D4F">{{ failTotal }}</div>
          </div>
          <div>
            <div class="text-gray-500 text-xs mb-1">总体成功率</div>
            <div class="font-medium">{{ overallRate }}%</div>
          </div>
        </div>
      </template>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import dayjs from 'dayjs'
import { getTestDetail } from '@/api/test'
import type { TestDetail } from '@/types/test'

const props = defineProps<{
  visible: boolean
  sessionId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

const loading = ref(false)
const detail = ref<TestDetail | null>(null)

watch(() => props.visible, async (val) => {
  if (val && props.sessionId) {
    loading.value = true
    try {
      detail.value = await getTestDetail(props.sessionId)
    } catch {
      detail.value = null
    } finally {
      loading.value = false
    }
  }
})

const totalMessages = computed(() => detail.value?.statistics.reduce((s, r) => s + r.totalMessages, 0) || 0)
const successTotal = computed(() => detail.value?.statistics.reduce((s, r) => s + r.successCount, 0) || 0)
const failTotal = computed(() => detail.value?.statistics.reduce((s, r) => s + r.messageLoss + r.decodeFail + r.invalidField, 0) || 0)
const overallRate = computed(() => {
  if (!totalMessages.value) return 0
  return ((successTotal.value / totalMessages.value) * 100).toFixed(1)
})

function formatTime(t: string) {
  return t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'
}
</script>
