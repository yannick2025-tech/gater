<template>
  <el-dialog
    v-model="visible"
    title="测试详情"
    width="820px"
    :close-on-click-modal="false"
    class="detail-dialog"
  >
    <div class="detail-body">
      <!-- Left: Stats -->
      <div class="stats-panel">
        <h4 class="panel-title">测试统计</h4>
        <div class="stat-grid">
          <div class="stat-item">
            <span class="stat-value">{{ stats.totalMessages }}</span>
            <span class="stat-label">总报文数</span>
          </div>
          <div class="stat-item">
            <span class="stat-value success">{{ stats.successTotal }}</span>
            <span class="stat-label">通过</span>
          </div>
          <div class="stat-item">
            <span class="stat-value danger">{{ stats.failTotal }}</span>
            <span class="stat-label">失败</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ formatDuration(stats.duration) }}</span>
            <span class="stat-label">耗时</span>
          </div>
        </div>
        <div class="stat-rate">
          <span>成功率：</span>
          <strong>{{ formatRate(stats.successRate) }}%</strong>
        </div>
      </div>

      <!-- Right: Summary Table -->
      <div class="summary-panel">
        <h4 class="panel-title">结果汇总</h4>
        <el-table
          :data="summaryRows"
          size="small"
          border
          style="width: 100%"
          max-height="340px"
          :header-cell-style="{ background: '#fafafa', color: '#666', fontWeight: '500', fontSize: '12px' }"
          empty-text="暂无数据"
        >
          <el-table-column label="#" width="40" type="index" />
          <el-table-column prop="funcCode" label="报文类型" min-width="90">
            <template #default="{ row }">
              <code class="func-code">{{ row.funcCode }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="direction" label="方向" width="90" align="center">
            <template #default="{ row }">
              <span :class="'dir-' + row.direction">{{ dirLabel(row.direction) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="结果" width="70" align="center">
            <template #default="{ row }">
              <el-tag
                :type="row.isSuccess ? 'success' : 'danger'"
                size="small"
                effect="light"
              >
                {{ row.isSuccess ? '通过' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button
                type="primary"
                link
                size="small"
                @click="viewMessage(row)"
              >
                查看报文
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- Footer -->
    <template #footer>
      <el-button @click="visible = false" size="small">关闭</el-button>
    </template>

    <!-- Message View Modal (receives data via prop) -->
    <MessageViewModal v-model="showMsgModal" :messages="msgFilterList" />
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getTestDetail, getMessageArchives } from '@/api/test'
import MessageViewModal from './MessageViewModal.vue'

const props = defineProps<{
  modelValue: boolean
  sessionId?: string   // 真实的 session ID (UUID格式)
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

// Data
const loading = ref(false)
const sessionIdRef = ref('')
const stats = ref({
  totalMessages: 0,
  successTotal: 0,
  failTotal: 0,
  duration: 0,
  successRate: 0,
})
const statistics = ref<any[]>([])
const allArchives = ref<any[]>([])
const showMsgModal = ref(false)

// Summary rows: deduplicated by (funcCode + direction + status), sorted by first timestamp
const summaryRows = computed(() => {
  const map = new Map<string, any>()

  // 遍历所有报文存档，按 (funcCode+direction+status) 分组，保留最近的一条
  for (const arch of allArchives.value) {
    const key = `${arch.funcCode}_${arch.direction}_${arch.status}`
    const existing = map.get(key)
    if (!existing || arch.timestamp > existing.timestamp) {
      map.set(key, {
        funcCode: arch.funcCode,
        direction: arch.direction,
        status: arch.status,
        isSuccess: arch.status === 'success',
        timestamp: arch.timestamp,
        hexData: arch.hexData,
        jsonData: arch.jsonData,
      })
    }
  }

  // 转为数组，按时间排序
  return Array.from(map.values()).sort((a, b) =>
    a.timestamp.localeCompare(b.timestamp)
  )
})

// Watch dialog open → fetch data using sessionId
watch(() => props.modelValue, async (val) => {
  if (!val) return

  // Use the real sessionId passed as prop
  const sid = props.sessionId
  if (!sid) return

  loading.value = true
  try {
    // Fetch test report detail by sessionId（可能返回404，如session还在运行中未入库）
    const detail = await getTestDetail(sid)
    statistics.value = detail.statistics || []

    // Compute stats
    let total = 0, success = 0, fail = 0
    for (const s of statistics.value) {
      total += s.totalMessages
      success += s.successCount
      fail += (s.decodeFail || 0) + (s.invalidField || 0) + (s.messageLoss || 0)
    }
    stats.value.totalMessages = total
    stats.value.successTotal = success
    stats.value.failTotal = fail
    stats.value.duration = 0
    stats.value.successRate = total > 0 ? Math.round(success / total * 100 * 10) / 10 : 0

    // Fetch all message archives for this session
    allArchives.value = []
    try { allArchives.value = await getMessageArchives(sid, '', '') } catch { /* empty */ }
    if (sessionIdRef.value) {
      try {
        allArchives.value = await getMessageArchives(sessionIdRef.value, '', '')
      } catch {
        allArchives.value = []
      }
    }
  } catch (e: any) {
    // 报告尚未入库（session仍在运行中）→ 显示友好提示而非报错
    statistics.value = []
    allArchives.value = []
    stats.value.totalMessages = 0
    stats.value.successTotal = 0
    stats.value.failTotal = 0
    stats.value.duration = 0
    stats.value.successRate = 0
  } finally {
    loading.value = false
  }
})

// Shared ref for passing filtered messages to MessageViewModal (module-level)
const msgFilterList = ref<any[]>([])

function viewMessage(row: any) {
  const matched = allArchives.value.filter(
    a => a.funcCode === row.funcCode && a.direction === row.direction && a.status === row.status
  )
  msgFilterList.value = matched.length > 0 ? matched : [row]
  showMsgModal.value = true
}

function formatDuration(ms: number): string {
  if (!ms) return '--'
  const sec = Math.floor(ms / 1000)
  const m = Math.floor(sec / 60)
  const s = sec % 60
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

function formatRate(rate: number): string {
  return rate ? rate.toFixed(1) : '0.0'
}

function dirLabel(dir: string): string {
  const labels: Record<string, string> = {
    '充电桩→平台': '↑ 上行',
    '平台→充电桩': '↓ 下行',
    '回复': '↔ 回复',
  }
  return labels[dir] || dir
}
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

.stat-value.success { color: #67c23a; }
.stat-value.danger { color: #f56c6c; }

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

.func-code {
  background: #f0f2f5;
  padding: 1px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 12px;
  color: #e6a23c;
  white-space: nowrap;
}

.dir-上行 { color: #67c23a; }
.dir-下行 { color: #409eff; }
.dir-回复 { color: #909399; }
</style>
