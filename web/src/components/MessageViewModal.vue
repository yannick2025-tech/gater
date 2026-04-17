<template>
  <el-dialog
    v-model="visible"
    title="查看报文"
    width="820px"
    :close-on-click-modal="false"
    class="msg-dialog"
  >
    <div class="filter-bar">
      <span class="msg-count">共 {{ displayMessages.length }} 条报文</span>
    </div>

    <!-- Message list -->
    <div class="message-list">
      <div v-for="(msg, idx) in displayMessages" :key="idx" class="message-item">
        <div class="msg-header">
          <el-tag :type="getDirType(msg.direction)" size="small" effect="light" round>
            {{ getDirLabel(msg.direction) }}
          </el-tag>
          <code class="func-code">{{ msg.funcCode }}</code>
          <span class="msg-time">{{ msg.timestamp }}</span>
          <el-tag :type="msg.status === 'success' ? 'success' : 'danger'" size="small" effect="light">
            {{ msg.status === 'success' ? '通过' : '失败' }}
          </el-tag>
          <el-button type="primary" link size="small" @click="copyContent(msg)">
            复制内容
          </el-button>
        </div>
        <div class="msg-body-row">
          <pre class="msg-hex"><code>{{ msg.hexData || '--' }}</code></pre>
          <pre class="msg-json"><code>{{ formatJson(msg.jsonData) }}</code></pre>
        </div>
        <div v-if="msg.errorMsg" class="error-msg">错误: {{ msg.errorMsg }}</div>
      </div>
      <div v-if="displayMessages.length === 0" class="empty-hint">暂无报文数据</div>
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
  messages?: any[]   // 从 TestDetailModal 通过 prop 传入的报文列表
}>()

const emit = defineEmits<{ 'update:modelValue': [val: boolean] }>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

// Use prop messages if provided, fallback to empty array
const displayMessages = computed(() => {
  const msgs = props.messages
  return Array.isArray(msgs) && msgs.length > 0 ? msgs : []
})

function formatJson(payload: string): string {
  if (!payload) return ''
  try { return JSON.stringify(JSON.parse(payload), null, 2) } catch { return payload }
}

async function copyContent(msg: any) {
  const text = `HEX:\n${msg.hexData || ''}\n\nJSON:\n${formatJson(msg.jsonData)}`
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text; document.body.appendChild(ta)
    ta.select(); document.execCommand('copy')
    document.body.removeChild(ta)
    ElMessage.success('已复制到剪贴板')
  }
}

function getDirType(dir: string): 'success' | 'primary' | 'info' | 'warning' | 'danger' | '' {
  if (dir.includes('充电桩→')) return 'success'
  if (dir.includes('平台→')) return 'primary'
  if (dir.includes('回复')) return 'info'
  return ''
}

function getDirLabel(dir: string): string {
  if (dir.includes('充电桩→')) return '上行 ↑'
  if (dir.includes('平台→')) return '下行 ↓'
  if (dir.includes('回复')) return '回复 ↔'
  return dir
}
</script>

<style scoped>
.filter-bar {
  margin-bottom: 10px;
}

.msg-count { font-size: 12px; color: #999; }

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
}
.message-item:last-child { margin-bottom: 0; }

.msg-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background-color: #f0f1f3;
  font-size: 12px;
  flex-wrap: wrap;
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

.msg-time {
  margin-left: auto;
  color: #aaa;
  font-family: monospace;
  font-size: 11px;
}

.msg-body-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  background-color: #1e1e1e;
  padding: 12px 14px;
  border-radius: 0 0 6px 6px;
}

.msg-hex,
.msg-json {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 11px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  color: #d4d4d4;
  max-height: 120px;
  min-width: 200px;
}

.error-msg {
  padding: 6px 14px;
  background-color: #fef0f0;
  color: #f56c6c;
  font-size: 11px;
  border-top: 1px solid #fde2e2;
}

.empty-hint { text-align: center; color: #ccc; padding: 40px 0; font-size: 13px; }
</style>
