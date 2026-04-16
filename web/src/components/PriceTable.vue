<template>
  <div>
    <div class="flex items-center justify-between mb-2">
      <label class="text-sm">
        价格时段表 <span class="text-red-500">*</span>
      </label>
      <div class="flex gap-2">
        <el-button size="small" :disabled="isLoading || priceRows.length >= 48" @click="addRow">
          <Plus class="w-4 h-4" />
        </el-button>
        <el-button size="small" :disabled="isLoading || priceRows.length <= 1" @click="deleteLastRow">
          <Minus class="w-4 h-4" />
        </el-button>
      </div>
    </div>

    <div class="border rounded overflow-auto max-h-96">
      <el-table :data="priceRows" stripe size="small">
        <el-table-column label="时段类型" width="130">
          <template #default="{ row }">
            <el-select v-model="row.periodType" placeholder="选择" :disabled="isLoading" size="small">
              <el-option label="尖峰" value="peak" />
              <el-option label="高峰" value="high" />
              <el-option label="平段" value="flat" />
              <el-option label="低谷" value="valley" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" width="140">
          <template #default="{ row }">
            <el-time-select v-model="row.startTime" :disabled="isLoading" size="small" start="00:00" end="23:59" step="00:30" />
          </template>
        </el-table-column>
        <el-table-column label="结束时间" width="140">
          <template #default="{ row }">
            <el-time-select v-model="row.endTime" :disabled="isLoading" size="small" start="00:00" end="23:59" step="00:30" />
          </template>
        </el-table-column>
        <el-table-column label="电费(元)" width="130">
          <template #default="{ row }">
            <el-input v-model="row.electricityFee" type="number" placeholder="0.0000" :disabled="isLoading" size="small" />
          </template>
        </el-table-column>
        <el-table-column label="服务费(元)" width="130">
          <template #default="{ row }">
            <el-input v-model="row.serviceFee" type="number" placeholder="0.0000" :disabled="isLoading" size="small" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="70" fixed="right">
          <template #default="{ $index }">
            <el-button
              type="danger"
              text
              size="small"
              :disabled="isLoading || priceRows.length <= 1"
              @click="deleteRow($index)"
            >
              <Minus class="w-4 h-4" />
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Plus, Minus } from 'lucide-vue-next'
import type { PriceRow } from '@/types/test'

const props = defineProps<{
  modelValue: PriceRow[]
  isLoading: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [rows: PriceRow[]]
}>()

const priceRows = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

function addRow() {
  if (priceRows.value.length < 48) {
    const lastRow = priceRows.value[priceRows.value.length - 1]
    priceRows.value = [
      ...priceRows.value,
      {
        id: Date.now().toString(),
        periodType: '',
        startTime: lastRow?.endTime || '00:00',
        endTime: '23:59',
        electricityFee: '',
        serviceFee: '',
      },
    ]
  }
}

function deleteLastRow() {
  if (priceRows.value.length > 1) {
    priceRows.value = priceRows.value.slice(0, -1)
  }
}

function deleteRow(index: number) {
  if (priceRows.value.length > 1) {
    priceRows.value = priceRows.value.filter((_, i) => i !== index)
  }
}
</script>
