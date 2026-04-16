<template>
  <div class="bg-white p-6 rounded-lg shadow mb-6">
    <h3 class="text-base font-semibold mb-4">测试配置</h3>

    <div class="space-y-4">
      <div>
        <label class="block mb-2 text-sm">
          请选择测试用例 <span class="text-red-500">*</span>
        </label>
        <el-select
          v-model="testCase"
          placeholder="请选择"
          :disabled="testLoading"
          class="w-full"
        >
          <el-option label="基础充电测试" value="basic_charging" />
          <el-option label="SFTP升级测试" value="sftp_upgrade" />
          <el-option label="平台下发配置测试" value="config_download" />
        </el-select>
      </div>

      <!-- 基础充电测试 -->
      <template v-if="testCase === 'basic_charging'">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block mb-2 text-sm">账户余额 <span class="text-red-500">*</span></label>
            <el-input v-model="basicForm.balance" type="number" placeholder="0.00" :disabled="testLoading">
              <template #suffix>元</template>
            </el-input>
          </div>
          <div>
            <label class="block mb-2 text-sm">停止码 <span class="text-red-500">*</span></label>
            <el-input
              v-model="basicForm.stopCode"
              placeholder="1234"
              maxlength="4"
              :disabled="testLoading"
              @input="basicForm.stopCode = basicForm.stopCode.replace(/\D/g, '').slice(0, 4)"
            />
          </div>
        </div>

        <div>
          <label class="block mb-2 text-sm">充电限制配额（互斥，填写任一项）</label>
          <div class="grid grid-cols-4 gap-4 p-4 bg-gray-50 rounded border border-gray-200">
            <div>
              <label class="block mb-1 text-xs text-gray-500">最大充电量</label>
              <el-input
                v-model="basicForm.maxCharge"
                type="number"
                placeholder="0"
                :disabled="testLoading || hasOtherConstraint('maxCharge')"
                @input="clearOtherConstraints('maxCharge')"
              >
                <template #suffix>kWh</template>
              </el-input>
            </div>
            <div>
              <label class="block mb-1 text-xs text-gray-500">时长</label>
              <el-input
                v-model="basicForm.duration"
                type="number"
                placeholder="0"
                :disabled="testLoading || hasOtherConstraint('duration')"
                @input="clearOtherConstraints('duration')"
              >
                <template #suffix>分钟</template>
              </el-input>
            </div>
            <div>
              <label class="block mb-1 text-xs text-gray-500">金额</label>
              <el-input
                v-model="basicForm.amount"
                type="number"
                placeholder="0"
                :disabled="testLoading || hasOtherConstraint('amount')"
                @input="clearOtherConstraints('amount')"
              >
                <template #suffix>元</template>
              </el-input>
            </div>
            <div>
              <label class="block mb-1 text-xs text-gray-500">SOC</label>
              <el-input
                v-model="basicForm.soc"
                type="number"
                placeholder="0"
                :disabled="testLoading || hasOtherConstraint('soc')"
                @input="clearOtherConstraints('soc')"
              >
                <template #suffix>%</template>
              </el-input>
            </div>
          </div>
        </div>

        <PriceTable v-model="basicForm.priceRows" :is-loading="testLoading" />
      </template>

      <!-- SFTP升级测试 -->
      <template v-if="testCase === 'sftp_upgrade'">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block mb-2 text-sm">固件版本 <span class="text-red-500">*</span></label>
            <el-input v-model="sftpForm.firmwareVersion" placeholder="请输入固件版本" :disabled="testLoading" />
          </div>
          <div>
            <label class="block mb-2 text-sm">SFTP地址</label>
            <el-input v-model="sftpForm.sftpAddr" placeholder="sftp://192.168.1.100/firmware.bin" :disabled="testLoading" />
          </div>
        </div>
      </template>

      <!-- 平台下发配置测试 -->
      <template v-if="testCase === 'config_download'">
        <div>
          <label class="block mb-2 text-sm">
            配置项 JSON <span class="text-red-500">*</span>
            <span class="text-gray-400 text-xs ml-2">支持功能码: 0xC2(配置下发)、0x22(计费规则)、0x0C(设备参数查询)</span>
          </label>
          <el-input
            v-model="configForm.jsonInput"
            type="textarea"
            :rows="8"
            placeholder='[
  {"funcCode": 194, "payload": {"paramList": [{"seq": 1, "valueBytes": [1, 2]}]}},
  {"funcCode": 34, "payload": {"feeNum": 1, "listFee": [{"hour": 8, "min": 0, "powerFee": 10000, "svcFee": 800, "type": 2, "limitedP": 0}]}}
]'
            :disabled="testLoading"
          />
          <div v-if="configForm.error" class="mt-1 text-xs text-red-500">{{ configForm.error }}</div>
        </div>
      </template>

      <!-- 测试进度 -->
      <div v-if="currentStatus && currentStatus.status === 'running'" class="p-4 bg-blue-50 rounded border border-blue-200">
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium">测试进行中...</span>
          <span class="text-sm text-blue-600">{{ currentStatus.stepName }}</span>
        </div>
        <el-progress :percentage="currentStatus.progress" :stroke-width="8" />
      </div>

      <!-- 开始测试按钮 -->
      <div v-if="testCase" class="flex justify-end pt-2">
        <el-button
          type="primary"
          :loading="testLoading"
          :disabled="!isOnline || !testCase"
          style="background-color: #148493; border-color: #148493"
          @click="handleStart"
        >
          {{ testLoading ? '测试中' : '开始测试' }}
        </el-button>
      </div>
      <div v-if="!isOnline && testCase" class="text-xs text-gray-400 text-right">设备未连接，无法开始测试</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import PriceTable from './PriceTable.vue'
import type { TestStatus, ConfigItem, PriceRow } from '@/types/test'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  gunNumber: string
  isOnline: boolean
  testLoading: boolean
  currentStatus: TestStatus | null
}>()

const emit = defineEmits<{
  startTest: [testCase: string, gunNumber: string]
  startConfig: [gunNumber: string, items: ConfigItem[]]
}>()

const testCase = ref<'' | 'basic_charging' | 'sftp_upgrade' | 'config_download'>('')

const basicForm = reactive({
  balance: '',
  stopCode: '1234',
  maxCharge: '',
  duration: '',
  amount: '',
  soc: '',
  priceRows: [
    { id: '1', periodType: '' as const, startTime: '00:00', endTime: '23:59', electricityFee: '', serviceFee: '' },
  ] as PriceRow[],
})

const sftpForm = reactive({
  firmwareVersion: '',
  sftpAddr: '',
})

const configForm = reactive({
  jsonInput: '',
  error: '',
})

const constraintFields = ['maxCharge', 'duration', 'amount', 'soc'] as const

function hasOtherConstraint(field: string) {
  return constraintFields.filter((f) => f !== field).some((f) => !!(basicForm as any)[f])
}

function clearOtherConstraints(field: string) {
  if ((basicForm as any)[field]) {
    constraintFields.forEach((f) => {
      if (f !== field) (basicForm as any)[f] = ''
    })
  }
}

function handleStart() {
  if (!props.gunNumber) {
    ElMessage.warning('请先查询设备')
    return
  }
  if (!props.isOnline) {
    ElMessage.warning('设备未连接')
    return
  }

  if (testCase.value === 'config_download') {
    const items = parseConfigJson()
    if (items) {
      emit('startConfig', props.gunNumber, items)
    }
    return
  }

  emit('startTest', testCase.value, props.gunNumber)
}

function parseConfigJson(): ConfigItem[] | null {
  configForm.error = ''
  const input = configForm.jsonInput.trim()
  if (!input) {
    configForm.error = '请输入配置项 JSON'
    return null
  }
  try {
    const parsed = JSON.parse(input)
    const arr = Array.isArray(parsed) ? parsed : [parsed]
    for (let i = 0; i < arr.length; i++) {
      if (!arr[i].funcCode || !arr[i].payload) {
        configForm.error = `item[${i}]: 缺少 funcCode 或 payload`
        return null
      }
      if (![194, 34, 12].includes(arr[i].funcCode)) {
        configForm.error = `item[${i}]: 不支持的功能码 0x${arr[i].funcCode.toString(16)} (支持: 0xC2, 0x22, 0x0C)`
        return null
      }
    }
    return arr as ConfigItem[]
  } catch (e: any) {
    configForm.error = 'JSON 格式错误: ' + e.message
    return null
  }
}
</script>
