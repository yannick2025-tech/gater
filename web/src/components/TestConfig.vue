<template>
  <div class="test-config-card">
    <!-- Header -->
    <div class="card-header">
      <h3 class="card-title">测试配置</h3>
    </div>

    <!-- Form（历史会话时通过CSS置灰只读，不用:disabled避免Element Plus内部状态闪烁） -->
    <el-form
      ref="formRef"
      :model="formData"
      label-width="0px"
      class="config-form"
      :class="{ 'form-disabled': isHistorical }"
      label-position="top"
    >
      <!-- 用例选择 -->
      <div class="form-section">
        <label class="required-label"><span class="label-asterisk">*</span> 请选择测试用例</label>
        <el-select
          v-model="formData.scenario"
          placeholder="请选择"
          class="full-select"
          size="default"
          :disabled="isCharging"
        >
          <el-option label="基础充电测试" value="basic_charging" />
          <el-option label="SFTP升级测试" value="sftp_upgrade" />
          <el-option label="配置下发测试" value="config_download" />
        </el-select>
      </div>

      <!-- 基础充电参数 (仅 basic_charging 显示) -->
      <template v-if="showChargingParams">
        <div class="form-section" :class="{ 'charging-disabled': isCharging }">
          <div class="section-label" v-if="isCharging">基础充电测试（充电中，禁止编辑）</div>
          <div class="form-row three-col">
            <el-form-item label="输出电压(V)" class="form-item-nested">
              <el-input-number v-model="formData.voltage" :min="0" :max="1000" controls-position="right" class="full-width" size="default" />
            </el-form-item>
            <el-form-item label="电流限流(A)" class="form-item-nested">
              <el-input-number v-model="formData.amperage" :min="0" :max="200" controls-position="right" class="full-width" size="default" />
            </el-form-item>
            <el-form-item label="充电模式" class="form-item-nested">
              <el-select v-model="formData.chargeMode" class="full-width">
                <el-option label="直流" value="DC" />
                <el-option label="交流" value="AC" />
              </el-select>
            </el-form-item>
          </div>

          <div class="form-row two-col">
            <el-form-item label="充电电量(kWh)" class="form-item-nested">
              <el-input-number v-model="formData.energy" :min="0" :max="500" :precision="2" controls-position="right" class="full-width" size="default" />
            </el-form-item>
            <el-form-item label="SOC目标(%)" class="form-item-nested">
              <el-input-number v-model="formData.targetSoc" :min="0" :max="100" controls-position="right" class="full-width" size="default" />
            </el-form-item>
          </div>

          <!-- 新增参数：VIN码 / 账户余额 / 屏显模式 -->
          <div class="form-row three-col">
            <el-form-item label="VIN码" class="form-item-nested">
              <el-input v-model="formData.vinCode" placeholder="17位车辆识别码(可选)" maxlength="17" />
            </el-form-item>
            <el-form-item label="账户余额(元)" class="form-item-nested">
              <el-input-number v-model="formData.balance" :min="0" :precision="2" controls-position="right" class="full-width" size="default">
                <template #append>元</template>
              </el-input-number>
            </el-form-item>
            <el-form-item label="屏显模式" class="form-item-nested">
              <el-select v-model="formData.displayMode" placeholder="请选择" class="full-width" clearable>
                <el-option label="模式0" value="0" />
                <el-option label="模式1" value="1" />
              </el-select>
            </el-form-item>
          </div>
        </div>

        <!-- 充电限制配额 -->
        <div class="quota-box">
          <div class="quota-title">充电限制配额</div>
          <div class="quota-body">
            <PriceTable v-model:prices="formData.prices" :disabled="isCharging" />
          </div>
        </div>
      </template>

      <!-- SFTP升级参数 -->
      <template v-if="formData.scenario === 'sftp_upgrade'">
        <div class="form-section">
          <div class="form-row two-col">
            <el-form-item label="SFTP服务器地址" class="form-item-nested">
              <el-input v-model="formData.sftpHost" placeholder="例: 192.168.1.100" />
            </el-form-item>
            <el-form-item label="端口" class="form-item-nested">
              <el-input-number v-model="formData.sftpPort" :min="1" :max="65535" controls-position="right" class="full-width" />
            </el-form-item>
          </div>
          <div class="form-row two-col">
            <el-form-item label="用户名" class="form-item-nested">
              <el-input v-model="formData.sftpUser" placeholder="SFTP用户名" />
            </el-form-item>
            <el-form-item label="密码" class="form-item-nested">
              <el-input v-model="formData.sftpPass" type="password" placeholder="SFTP密码" show-password />
            </el-form-item>
          </div>
          <el-form-item label="固件文件路径" class="form-item-nested full-row">
            <el-input v-model="formData.firmwarePath" placeholder="/firmware/gater_v1.6.0.bin" />
          </el-form-item>
          <el-form-item label="校验值(MD5)" class="form-item-nested full-row">
            <el-input v-model="formData.md5Checksum" placeholder="32位MD5哈希值" />
          </el-form-item>
        </div>
      </template>

      <!-- 配置下发参数 -->
      <template v-if="formData.scenario === 'config_download'">
        <div class="form-section">
          <el-form-item label="配置项(funcCode)" class="form-item-nested full-row">
            <el-select v-model="formData.configFuncCode" placeholder="选择功能码" class="full-width" @change="onConfigFuncCodeChange">
              <el-option label="0xC2 - 配置信息下发" value="0xC2" />
              <el-option label="0x22 - 分时段计费规则下发" value="0x22" />
              <el-option label="0x0C - 设备参数查询" value="0x0C" />
            </el-select>
          </el-form-item>
          <el-form-item label="Payload内容(JSON)" class="form-item-nested full-row">
            <el-input
              v-model="formData.configPayload"
              type="textarea"
              :rows="6"
              placeholder='选择配置项后自动填充默认JSON，可修改参数值'
              class="json-textarea"
            />
          </el-form-item>
          <div v-if="payloadError" class="error-hint">{{ payloadError }}</div>
        </div>
      </template>

      <!-- Footer：历史/活跃会话不同提示 -->
      <div class="card-footer">
        <span v-if="isHistorical" class="offline-hint">该会话已完成，仅可查看配置</span>
        <span v-else-if="!canStartTest && !isCharging" class="offline-hint">请选择一个活跃会话</span>
        <el-button
          v-if="!isCharging"
          type="primary"
          size="large"
          class="start-btn"
          :disabled="!canStartTest || isHistorical"
          @click="handleStart"
        >
          {{ isHistorical ? '已结束' : '开始测试' }}
        </el-button>
        <el-button
          v-else
          type="danger"
          size="large"
          class="start-btn"
          :disabled="isStopDisabled"
          @click="handleStop"
        >
          {{ isStopDisabled ? '已结束' : '结束充电' }}
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import PriceTable, { type PriceRow } from './PriceTable.vue'

const props = defineProps<{
  /** 是否允许开始测试（基于是否选中了活跃会话） */
  canStartTest?: boolean
  /** 是否为历史会话（只读模式，不可配置） */
  isHistorical?: boolean
  /** 是否正在充电中 */
  isCharging?: boolean
  /** 充电是否已停止（0x05已收到） */
  isChargingStopped?: boolean
}>()

const emit = defineEmits<{
  (e: 'start', data: Record<string, unknown>): void
  (e: 'configStart', gunNumber: string, items: Array<{ funcCode: number; payload: Record<string, unknown> }>): void
  (e: 'stop'): void
}>()

const isStopDisabled = computed(() => props.isChargingStopped === true)

const formRef = ref<FormInstance>()

const formData = ref({
  scenario: '',
  voltage: 200,
  amperage: 32,
  chargeMode: 'DC',
  energy: 50,
  targetSoc: 95,
  vinCode: '' as string,        // VIN码（可选，17位）
  balance: 500.00 as number,    // 账户余额（必填，单位元）
  displayMode: '' as string,   // 屏显模式（可选，0或1）
  prices: [] as PriceRow[],
  sftpHost: '',
  sftpPort: 22,
  sftpUser: '',
  sftpPass: '',
  firmwarePath: '',
  md5Checksum: '',
  configFuncCode: '',
  configPayload: '',
})

// 配置项默认 JSON payload（基于协议文档 appendix.MD 附录-配置项列表）
const configPayloadDefaults: Record<string, string> = {
  '0xC2': JSON.stringify({
    paramList: [
      { seq: 6, value: "192.168.1.100" },
      { seq: 7, value: "8888" },
      { seq: 10, value: "192.168.0.1" },
      { seq: 11, value: "255.255.255.0" },
      { seq: 12, value: "192.168.0.1" },
      { seq: 14, value: "20260101120000" }
    ]
  }, null, 2),
  '0x22': JSON.stringify({
    feeNum: 1,
    listFee: [
      { hour: 0, min: 0, powerFee: 10000, svcFee: 10000, type: 2, limitedP: 0 }
    ]
  }, null, 2),
  '0x0C': JSON.stringify({
    cmdCode: 0,
    data: [6, 7, 10, 14]
  }, null, 2),
}

function onConfigFuncCodeChange(val: string) {
  if (configPayloadDefaults[val]) {
    formData.value.configPayload = configPayloadDefaults[val]
  }
}

const showChargingParams = computed(() => formData.value.scenario === 'basic_charging')

// 切换到基础充电时初始化默认时段行
watch(() => formData.value.scenario, (val) => {
  if (val === 'basic_charging' && formData.value.prices.length === 0) {
    formData.value.prices = [
      { startTime: '00:00', endTime: '23:59', electricityFee: 0, serviceFee: 0, peakValleyType: 2 },
    ]
  }
})

const payloadError = computed(() => {
  if (formData.value.scenario !== 'config_download') return ''
  const payload = formData.value.configPayload?.trim()
  if (!payload) return ''
  try {
    JSON.parse(payload)
    return ''
  } catch {
    return 'JSON格式错误，请检查语法'
  }
})

function handleStart() {
  if (!props.canStartTest) {
    ElMessage.warning('请选择一个活跃会话')
    return
  }
  if (!formData.value.scenario) {
    ElMessage.warning('请先选择测试用例')
    return
  }
  if (payloadError.value) {
    ElMessage.error('请修正JSON格式')
    return
  }

  const scenario = formData.value.scenario

  // 配置下发：使用专用接口，只发送配置相关参数
  if (scenario === 'config_download') {
    const funcCodeStr = formData.value.configFuncCode
    if (!funcCodeStr) {
      ElMessage.error('请选择配置项(funcCode)')
      return
    }
    // 解析 funcCode: "0xC2" → 194
    const funcCode = parseInt(funcCodeStr, 16)
    if (isNaN(funcCode)) {
      ElMessage.error('funcCode格式错误')
      return
    }
    const payloadStr = formData.value.configPayload?.trim()
    if (!payloadStr) {
      ElMessage.error('请填写Payload内容')
      return
    }
    let payloadObj: Record<string, unknown>
    try {
      payloadObj = JSON.parse(payloadStr)
    } catch {
      ElMessage.error('Payload JSON格式错误')
      return
    }
    emit('configStart', '', [{ funcCode, payload: payloadObj }])
    return
  }

  // 基础充电参数校验
  if (scenario === 'basic_charging') {
    // VIN码：可选，但填了必须是17位
    const vin = formData.value.vinCode?.trim()
    if (vin && vin.length !== 17) {
      ElMessage.error('VIN码必须为17位')
      return
    }

    // 账户余额：必填
    if (formData.value.balance === null || formData.value.balance === undefined || formData.value.balance < 0) {
      ElMessage.error('请输入账户余额')
      return
    }

    // 只发送充电相关参数
    emit('start', {
      scenario,
      vinCode: formData.value.vinCode,
      balance: formData.value.balance,
      displayMode: formData.value.displayMode,
      targetSoc: formData.value.targetSoc,
      energy: formData.value.energy,
      prices: formData.value.prices,
    })
    return
  }

  // SFTP升级：只发送SFTP相关参数
  if (scenario === 'sftp_upgrade') {
    emit('start', {
      scenario,
      sftpHost: formData.value.sftpHost,
      sftpPort: formData.value.sftpPort,
      sftpUser: formData.value.sftpUser,
      sftpPass: formData.value.sftpPass,
      firmwarePath: formData.value.firmwarePath,
      md5Checksum: formData.value.md5Checksum,
    })
    return
  }

  emit('start', { ...formData.value })
}

function handleStop() {
  emit('stop')
}
</script>

<style scoped>
.test-config-card {
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

.config-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.required-label {
  font-size: 14px;
  font-weight: 500;
  color: #333;
  display: inline-block;
  line-height: 1.4;
}

.label-asterisk {
  color: #f56c6c;
  margin-right: 2px;
}

.full-select {
  width: 100%;
}

.form-row {
  display: flex;
  gap: 16px;
}

.form-row.three-col .form-item-nested {
  flex: 1;
  min-width: 0; /* 防止flex子项溢出 */
}

.form-row.two-col .form-item-nested {
  flex: 1;
  min-width: 0;
}

.full-row {
  width: 100%;
}

.form-item-nested {
  margin-bottom: 0 !important;
  min-width: 0;
}

.form-item-nested :deep(.el-form-item__content) {
  justify-content: flex-start;
}

.form-item-nested :deep(.el-form-item__label) {
  font-size: 13px;
  padding: 0 0 4px;
  line-height: 1.4;
  white-space: nowrap;
}

.full-width {
  width: 100% !important;
}

/* 充电限制配额 */
.quota-box {
  background-color: #F7F8FA;
  border-radius: 6px;
  overflow: hidden;
}

.quota-title {
  padding: 12px 16px;
  font-size: 14px;
  font-weight: 500;
  color: #555;
  border-bottom: 1px solid #eee;
}

.quota-body {
  padding: 16px;
}

.json-textarea :deep(textarea) {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
}

.error-hint {
  color: #f56c6c;
  font-size: 12px;
  line-height: 1.4;
}

/* Footer */
.card-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px dashed #eee;
  margin-top: 4px;
}

.offline-hint {
  font-size: 13px;
  color: #e6a23c;
}

.start-btn {
  min-width: 120px;
  border-radius: 6px;
  font-size: 14px;
  padding: 10px 28px;
}

/* 历史会话时表单置灰只读 — 用CSS阻断交互，不用el-form的:disabled（会导致Element Plus内部状态闪烁） */
.form-disabled {
  opacity: 0.6;
  pointer-events: none;
  user-select: none;
}

/* 充电中时基础参数区域置灰（同上，仅针对基础充电测试区域） */
.charging-disabled {
  opacity: 0.55;
  pointer-events: none;
  user-select: none;
}

.section-label {
  font-size: 12px;
  color: #e6a23c;
  margin-bottom: 8px;
  font-weight: 500;
}

.form-disabled :deep(.el-input__inner),
.form-disabled :deep(.el-textarea__inner) {
  background-color: #f5f7fa;
  color: #999;
}
</style>
