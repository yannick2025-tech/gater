<template>
  <div class="test-results-card">
    <!-- Header -->
    <div class="card-header">
      <h3 class="card-title">测试结果</h3>
    </div>

    <!-- Table -->
    <el-table
      :data="results"
      stripe
      style="width: 100%"
      :header-cell-style="{ background: '#fafafa', color: '#666', fontWeight: '500' }"
      empty-text="暂无测试结果"
    >
      <el-table-column prop="protocolName" label="测试用例名称" min-width="160" />
      <el-table-column label="测试结果" width="100" align="center">
        <template #default="{ row }">
          <el-tag
            :type="row.isPass ? 'success' : 'danger'"
            effect="light"
            round
          >
            {{ row.isPass ? '通过' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="startTime" label="测试时间" min-width="170" />
      <el-table-column label="操作" width="120" align="right">
        <template #default="{ row }">
          <el-button type="primary" link @click="$emit('view-detail', row.id)">
            查看详情
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Footer -->
    <div class="card-footer">
      <div class="footer-left">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          layout="prev, next"
          small
          @current-change="handlePageChange"
        />
      </div>
      <div class="footer-right">
        <el-button plain size="small" @click="$emit('export')">
          <el-icon><Document /></el-icon>
          导出测试报告
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Document } from '@element-plus/icons-vue'

defineProps<{
  results: Array<Record<string, any>>
}>()

const emit = defineEmits<{
  'view-detail': [id: number]
  export: []
}>()

const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(2)

function handlePageChange(page: number) {
  // TODO: load page data
}
</script>

<style scoped>
.test-results-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  padding: 24px;
}

.card-header {
  margin-bottom: 16px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin: 0;
}

/* Table overrides */
.test-results-card :deep(.el-table) {
  --el-table-border-color: #f0f0f0;
}

.test-results-card :deep(.el-table th.el-table__cell) {
  border-bottom: none;
  font-size: 13px;
}

.test-results-card :deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid #f5f5f5;
  font-size: 13px;
}

.test-results-card :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: #fafbfc;
}

/* Footer */
.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 16px;
}

.footer-left {
  display: flex;
  gap: 12px;
}

.footer-right {
  display: flex;
  gap: 8px;
}
</style>
