<template>
  <div class="device-info-bar">
    <!-- 模式1：会话选择列表（未选中任何会话时显示） -->
    <div v-if="effectiveMode === 'select'" class="session-select-mode">
      <!-- Header -->
      <div class="select-header">
        <h3 class="card-title">会话列表</h3>
        <el-button plain size="small" @click="$emit('refresh-sessions')">
          刷新
        </el-button>
      </div>

      <!-- 会话表格 -->
      <el-table
        :data="sessions"
        stripe
        style="width: 100%"
        :header-cell-style="{ background: '#fafafa', color: '#666', fontWeight: '500' }"
        empty-text="暂无可用会话（等待充电桩连接...）"
        highlight-current-row
        @row-click="handleRowClick"
      >
        <el-table-column prop="gunNumber" label="桩编号" min-width="140" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isOnline" type="success" effect="light" size="small" round>在线</el-tag>
            <el-tag v-else-if="row.authState === 'pending'" type="warning" effect="light" size="small" round>认证中</el-tag>
            <el-tag v-else type="info" effect="light" size="small" round>{{ row.authState }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="connectedAt" label="连接时间" min-width="160" />
        <el-table-column prop="lastActive" label="最后活跃" min-width="160" />
        <el-table-column label="操作" width="100" align="center">
          <template #default="{ row }">
            <el-button
              v-if="row.isOnline"
              type="primary"
              link
              size="small"
              @click.stop="handleSelectSession(row)"
            >
              选择
            </el-button>
            <el-button
              v-else
              type="primary"
              link
              size="small"
              @click.stop="handleSelectSession(row)"
            >
              查看
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="select-footer-hint">
        <span class="hint-text">请选择一个会话以继续操作。在线会话可进行测试，离线/历史会话仅可查看报告。</span>
      </div>
    </div>

    <!-- 模式2：设备信息栏（已选中会话后显示） -->
    <template v-else>
      <div class="info-left">
        <span class="info-item">
          <label>桩编号：</label>{{ gunNumber || '--' }}
        </span>
        <div class="info-divider"></div>
        <span class="info-item">
          <label>协议名称：</label>{{ protocolName || '--' }}
        </span>
        <div class="info-divider"></div>
        <span class="info-item">
          <label>协议版本：</label>{{ protocolVersion || '--' }}
        </span>
        <div class="info-divider"></div>
        <el-tag v-if="isOnline" type="success" size="small" effect="light" round>在线</el-tag>
        <el-tag v-if="!isOnline && gunNumber" type="info" size="small" effect="light" round>历史</el-tag>
      </div>

      <div class="info-right">
        <!-- 返回选择列表 -->
        <el-button plain size="small" class="back-btn" @click="$emit('back-to-list')">
          重新选择
        </el-button>

        <!-- 断开按钮（活跃会话可点击，历史置灰） -->
        <el-button
          type="danger"
          class="disconnect-btn"
          :disabled="!isOnline"
          :plain="!isOnline"
          @click="$emit('disconnect')"
        >
          {{ isOnline ? '断开连接' : '历史会话' }}
        </el-button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SessionItem } from '@/types/device'

const props = defineProps<{
  gunNumber: string
  isOnline?: boolean
  protocolName?: string
  protocolVersion?: string
  sessions?: SessionItem[]
  /** 显示模式：'select'=会话列表, 'detail'=设备信息栏 */
  mode?: 'select' | 'detail'
}>()

const emit = defineEmits<{
  'select-session': [session: SessionItem]
  disconnect: []
  'refresh-sessions': []
  'back-to-list': []
}>()

// 默认根据是否有gunNumber判断模式；也可由外部显式控制
const effectiveMode = computed(() => props.mode || (props.gunNumber ? 'detail' : 'select'))

function handleRowClick(row: SessionItem) {
  // 行点击也触发选择
}

function handleSelectSession(session: SessionItem) {
  emit('select-session', session)
}
</script>

<style scoped>
.device-info-bar {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

/* ========== 模式1：会话选择列表 ========== */
.session-select-mode {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.select-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin: 0;
}

.session-select-mode :deep(.el-table) {
  --el-table-border-color: #f0f0f0;
}

.session-select-mode :deep(.el-table th.el-table__cell) {
  border-bottom: none;
  font-size: 13px;
}

.session-select-mode :deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid #f5f5f5;
  font-size: 13px;
  cursor: pointer;
}

.session-select-mode :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: #fafbfc;
}

.session-select-mode :deep(.el-table .current-row > td) {
  background-color: #ecf5ff !important;
}

.select-footer-hint {
  padding: 8px 4px;
}

.hint-text {
  font-size: 12px;
  color: #999;
  line-height: 1.6;
}

/* ========== 模式2：设备信息栏 ========== */
.device-info-bar:not(:has(.session-select-mode)) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
}

.info-left {
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-item {
  font-size: 14px;
  color: #333;
  white-space: nowrap;
}

.info-item label {
  color: #999;
  margin-right: 2px;
}

.info-divider {
  width: 1px;
  height: 14px;
  background-color: #ddd;
  margin: 0 12px;
}

.info-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.back-btn {
  border-radius: 6px;
}

.disconnect-btn {
  border-radius: 6px;
  padding: 8px 20px;
  font-size: 13px;
}
</style>
