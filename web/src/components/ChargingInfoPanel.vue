<template>
  <div class="charging-info-card">
    <div class="card-header">
      <h3 class="card-title">充电信息</h3>
    </div>

    <div class="info-grid">
      <!-- 时段费用 -->
      <div class="info-section">
        <h4 class="section-title">时段费率</h4>
        <el-table :data="info.prices || []" size="small" border>
          <el-table-column prop="startTime" label="开始时间" width="100" />
          <el-table-column prop="endTime" label="结束时间" width="100" />
          <el-table-column prop="electricityFee" label="电费(元/kWh)" width="120" />
          <el-table-column prop="serviceFee" label="服务费(元/kWh)" width="120" />
        </el-table>
      </div>

      <!-- 充电信息 -->
      <div class="info-section" v-if="info.chargingInfo">
        <h4 class="section-title">充电详情</h4>
        <div class="detail-grid">
          <div class="detail-item">
            <span class="label">平台充电开始时间</span>
            <span class="value">{{ info.chargingInfo.platformStartTime || '--' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">充电桩第1个0x06时间</span>
            <span class="value">{{ info.chargingInfo.firstDataTime || '--' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">充电桩充电开始时间</span>
            <span class="value">{{ info.chargingInfo.chargerStartTime || '--' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">订单号</span>
            <span class="value">{{ info.chargingInfo.chargingOrderNo || '--' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">充电时长</span>
            <span class="value">{{ formatDuration(info.chargingInfo.chargingDurationSec) }}</span>
          </div>
          <div class="detail-item">
            <span class="label">充电SOC</span>
            <span class="value">{{ info.chargingInfo.isChargingStopped ? info.chargingInfo.stopSOC : info.chargingInfo.currentSOC }}%</span>
          </div>
          <div class="detail-item">
            <span class="label">当前电量</span>
            <span class="value">{{ info.chargingInfo.currentElec?.toFixed(4) || '--' }} kWh</span>
          </div>
          <div class="detail-item">
            <span class="label">0x06上报次数</span>
            <span class="value">{{ info.chargingInfo.chargingDataCount }}</span>
          </div>
          <div class="detail-item">
            <span class="label">最后0x06时间</span>
            <span class="value">{{ info.chargingInfo.lastDataTime || '--' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">平台充电结束时间</span>
            <span class="value">{{ info.chargingInfo.platformStopTime || '--' }}</span>
          </div>
          <div class="detail-item">
            <span class="label">充电桩充电结束时间</span>
            <span class="value">{{ info.chargingInfo.chargerStopTime || '--' }}</span>
          </div>
        </div>
      </div>

      <!-- 校验结果 -->
      <div class="info-section" v-if="info.validationSummary">
        <h4 class="section-title">校验结果</h4>
        <div class="validation-summary">
          <el-tag type="success">通过: {{ info.validationSummary.pass }}</el-tag>
          <el-tag type="danger" v-if="info.validationSummary.fail > 0">失败: {{ info.validationSummary.fail }}</el-tag>
          <el-tag type="info">总计: {{ info.validationSummary.total }}</el-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  info: any
}>()

function formatDuration(sec: number | undefined): string {
  if (!sec && sec !== 0) return '--'
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}分${s}秒`
}
</script>

<style scoped>
.charging-info-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  padding: 24px;
}

.card-header {
  margin-bottom: 20px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin: 0;
}

.info-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.info-section {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #606266;
  margin: 0 0 12px 0;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px 24px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.detail-item .label {
  color: #909399;
  min-width: 160px;
  text-align: right;
}

.detail-item .value {
  color: #303133;
  font-weight: 500;
}

.validation-summary {
  display: flex;
  gap: 12px;
}
</style>
