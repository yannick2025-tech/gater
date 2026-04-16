<template>
  <el-dialog
    v-model="visible"
    title="查看测试结果中的报文"
    width="800px"
    :close-on-click-modal="false"
    class="msg-dialog"
  >
    <!-- Filter bar -->
    <div class="filter-bar">
      <div class="filter-left">
        <span class="filter-label">筛选：</span>
        <el-select v-model="filterType" size="small" class="filter-select" placeholder="全部类型">
          <el-option label="全部类型" value="" />
          <el-option label="请求报文" value="request" />
          <el-option label="响应报文" value="response" />
        </el-select>
      </div>
      <div class="filter-right">
        <span class="msg-count">共 {{ filteredMessages.length }} 条</span>
      </div>
    </div>

    <!-- Message list -->
    <div class="message-list">
      <div
        v-for="(msg, idx) in filteredMessages"
        :key="idx"
        class="message-item"
      >
        <div class="msg-header">
          <span class="msg-index">#{{ idx + 1 }}</span>
          <el-tag
            :type="msg.direction === 'request' ? '' : 'success'"
            size="small"
            effect="light"
          >
            {{ msg.direction === 'request' ? '请求' : '响应' }}
          </el-tag>
          <span class="msg-type">{{ msg.messageType }}</span>
          <span class="msg-time">{{ msg.timestamp }}</span>
          <el-button
            type="primary"
            link
            size="small"
            class="copy-btn"
            @click="copyContent(msg)"
          >
            复制内容
          </el-button>
        </div>
        <pre class="msg-body"><code>{{ formatJson(msg.payload) }}</code></pre>
      </div>

      <div v-if="filteredMessages.length === 0" class="empty-hint">
        暂无报文数据
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const filterType = ref('')

// Mock messages - replace with real data
const allMessages = ref([
  {
    direction: 'request',
    messageType: 'BootNotification',
    timestamp: '2026-03-05 14:30:01',
    payload: '[2,"19210001","BootNotification",{"chargePointVendor":"NTS","chargePointModel":"GaterV160","firmwareVersion":"v1.6.0"}]',
  },
  {
    direction: 'response',
    messageType: 'BootNotification',
    timestamp: '2026-03-05 14:30:01',
    payload: '[3,"19210001",{"status":"Accepted","currentTime":"2026-03-05T14:30:01Z","interval":300}]',
  },
  {
    direction: 'request',
    messageType: 'Heartbeat',
    timestamp: '2026-03-05 14:35:01',
    payload: '[4,"19210001","Heartbeat",{}]',
  },
  {
    direction: 'response',
    messageType: 'Heartbeat',
    timestamp: '2026-03-05 14:35:01',
    payload: '[5,"19210001",{"currentTime":"2026-03-05T14:35:01Z"}]',
  },
])

const filteredMessages = computed(() => {
  if (!filterType.value) return allMessages.value
  return allMessages.value.filter(m => m.direction === filterType.value)
})

function formatJson(payload: string): string {
  try {
    const obj = JSON.parse(payload)
    return JSON.stringify(obj, null, 2)
  } catch {
    return payload
  }
}

async function copyContent(msg: { payload: string }) {
  try {
    await navigator.clipboard.writeText(formatJson(msg.payload))
    ElMessage.success('已复制到剪贴板')
  } catch {
    // fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = formatJson(msg.payload)
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    ElMessage.success('已复制到剪贴板')
  }
}
</script>

<style scoped>
.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.filter-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-label {
  font-size: 13px;
  color: #666;
}

.filter-select {
  width: 140px;
}

.msg-count {
  font-size: 12px;
  color: #999;
}

.message-list {
  max-height: 450px;
  overflow-y: auto;
  border: 1px solid #eee;
  border-radius: 6px;
  padding: 8px;
}

.message-item {
  margin-bottom: 10px;
  background-color: #fafbfc;
  border-radius: 6px;
  overflow: hidden;
}

.message-item:last-child {
  margin-bottom: 0;
}

.msg-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background-color: #f0f1f3;
  font-size: 12px;
}

.msg-index {
  color: #999;
  font-weight: 500;
}

.msg-type {
  font-weight: 500;
  color: #333;
}

.msg-time {
  margin-left: auto;
  color: #aaa;
  font-family: monospace;
  font-size: 11px;
}

.copy-btn {
  font-size: 11px;
}

.msg-body {
  margin: 0;
  padding: 12px 14px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  font-size: 12px;
  line-height: 1.5;
  max-height: 200px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  border-radius: 0 0 6px 6px;
}

.empty-hint {
  text-align: center;
  color: #ccc;
  padding: 40px 0;
  font-size: 13px;
}
</style>
