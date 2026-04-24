<template>
  <div class="price-table-wrapper">
    <table class="price-table">
      <thead>
        <tr>
          <th>时段开始</th>
          <th>时段结束</th>
          <th>电费(元/kWh)</th>
          <th>服务费(元/kWh)</th>
          <th width="40"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, idx) in prices" :key="idx">
          <!-- 时段开始：第一行固定00:00，其他自动=上一行结束 -->
          <td class="time-cell">
            <el-time-select
              :model-value="row.startTime"
              start="00:00"
              step="01:00"
              end="23:59"
              placeholder="--:--"
              size="small"
              :disabled="true"
            />
          </td>
          <!-- 时段结束：支持选择 + 手动输入任意时间 -->
          <td class="time-cell">
            <el-time-picker
              :model-value="row.endTime"
              format="HH:mm"
              value-format="HH:mm"
              placeholder="--:--"
              size="small"
              :clearable="false"
              @change="(v: string | undefined) => onEndTimeChange(idx, v)"
            />
          </td>
          <!-- 电费 - 精度4位小数 -->
          <td>
            <el-input-number
              :model-value="row.electricityFee"
              :min="0"
              :precision="4"
              controls-position="right"
              size="small"
              style="width: 130px"
              @change="(v: number | undefined) => updateFee(idx, 'electricityFee', v)"
            />
          </td>
          <!-- 服务费 - 精度4位小数 -->
          <td>
            <el-input-number
              :model-value="row.serviceFee"
              :min="0"
              :precision="4"
              controls-position="right"
              size="small"
              style="width: 130px"
              @change="(v: number | undefined) => updateFee(idx, 'serviceFee', v)"
            />
          </td>
          <td class="action-col">
            <el-button
              type="danger"
              link
              size="small"
              :disabled="prices.length <= 1"
              @click="removeRow(idx)"
            >
              删除
            </el-button>
          </td>
        </tr>
        <tr v-if="prices.length === 0">
          <td colspan="5" class="empty-row">暂无配置，点击下方按钮添加</td>
        </tr>
      </tbody>
    </table>

    <div class="add-row-btn">
      <el-button
        type="primary" plain size="small"
        :disabled="!canAddRow || prices.length >= MAX_ROWS"
        @click="addRow"
      >
        + 添加时段
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">

import { computed } from 'vue'

const MAX_ROWS = 48

export interface PriceRow {
  startTime: string
  endTime: string
  electricityFee: number
  serviceFee: number
}

const props = defineProps<{
  prices: PriceRow[]
}>()

const emit = defineEmits<{
  'update:prices': [val: PriceRow[]]
}>()

/**
 * 核心规则：
 * 1. 默认一行：00:00 ~ 23:59
 * 2. 新增条件：最后一行的 endTime != "23:59"
 * 3. 新增行：startTime = 上行endTime，endTime = 23:59
 * 4. 删除后：最后一行 endTime 自动补为 23:59
 * 5. 全部时段必须连续覆盖 00:00 ~ 23:59，不重复不中断
 */

// 能否新增：最后一行结束时间不是23:59
const canAddRow = computed(() => {
  if (props.prices.length >= MAX_ROWS) return false
  if (props.prices.length === 0) return true
  const last = props.prices[props.prices.length - 1]
  return !!last && last.endTime !== '23:59'
})

function addRow() {
  if (!canAddRow.value) return

  const lastRow = props.prices[props.prices.length - 1]
  const newStart = lastRow ? lastRow.endTime : '00:00'
  const newRow: PriceRow = {
    startTime: newStart,
    endTime: '23:59',
    electricityFee: 0,
    serviceFee: 0,
  }
  emit('update:prices', [...props.prices, newRow])
}

function removeRow(idx: number) {
  if (props.prices.length <= 1) return

  const newPrices: PriceRow[] = []
  for (let i = 0; i < props.prices.length; i++) {
    if (i !== idx) {
      newPrices.push({ ...props.prices[i] })
    }
  }

  // 删除后末行必须以23:59结尾（保持完整覆盖）
  if (newPrices.length > 0) {
    newPrices[newPrices.length - 1].endTime = '23:59'
  }

  emit('update:prices', newPrices)
}

function onEndTimeChange(idx: number, val: string | undefined) {
  if (!val) return

  // 校验输入格式：必须为 HH:mm
  if (!/^\d{1,2}:\d{2}$/.test(val)) {
    return
  }

  // 校验：结束时间不能早于或等于开始时间
  const startMin = timeToMinutes(props.prices[idx].startTime)
  const endMin = timeToMinutes(val)
  if (endMin <= startMin) {
    return
  }

  const newPrices = props.prices.map((row, i) =>
    i === idx ? { ...row, endTime: val } : { ...row }
  )

  // 修改某行结束时，下一行（如果有）的开始时间自动衔接
  if (idx < newPrices.length - 1) {
    newPrices[idx + 1] = { ...newPrices[idx + 1], startTime: val }

    // 如果下一行的新开始时间 >= 它自己的结束时间，则自动调整下一行结束时间
    if (timeToMinutes(val) >= timeToMinutes(newPrices[idx + 1].endTime)) {
      newPrices[idx + 1] = { ...newPrices[idx + 1], endTime: '23:59' }
    }
  }

  emit('update:prices', newPrices)
}

function updateFee(idx: number, field: 'electricityFee' | 'serviceFee', val: number | undefined) {
  const newPrices = props.prices.map((row, i) =>
    i === idx ? { ...row, [field]: val ?? 0 } : { ...row }
  )
  emit('update:prices', newPrices)
}

/** "HH:mm" -> 分钟数 */
function timeToMinutes(t: string): number {
  if (!t) return 0
  const parts = t.split(':')
  return (parseInt(parts[0], 10) || 0) * 60 + (parseInt(parts[1], 10) || 0)
}
</script>

<style scoped>
.price-table-wrapper {
  font-size: 13px;
}

.price-table {
  width: 100%;
  border-collapse: collapse;
}

.price-table th {
  text-align: left;
  padding: 8px 10px;
  color: #666;
  font-weight: 500;
  border-bottom: 1px solid #eee;
  font-size: 12px;
}

.price-table td {
  padding: 8px 4px;
  vertical-align: middle;
}

.time-cell .el-select,
.time-cell .el-date-editor {
  width: 110px !important;
}

.action-col {
  text-align: center;
}

.empty-row {
  text-align: center;
  color: #ccc;
  padding: 16px !important;
}

.add-row-btn {
  margin-top: 8px;
}
</style>
