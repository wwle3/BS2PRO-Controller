<template>
  <div class="p-8 flex flex-col h-full space-y-6 overflow-y-auto">
    <header>
      <h2 class="text-xl font-bold tracking-tight text-slate-800 dark:text-white">设备参数设置</h2>
    </header>

    <div class="grid grid-cols-1 gap-6">
      <!-- 自动化逻辑 -->
      <div class="grid grid-cols-2 gap-6">
        <div class="p-6 rounded-[2.5rem] border space-y-6 surface-card">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-bold text-slate-700 dark:text-slate-300">智能变频</h3>
              <p class="text-[10px] text-slate-400">根据温度曲线自动调节风扇转速</p>
            </div>
            <LedToggle :active="config.autoControl" @click="handleAutoControlToggle"
                       :disabled="!isConnected" />
          </div>
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm font-bold text-slate-700 dark:text-slate-300">自动曲线偏移</h3>
              <p class="text-[10px] text-slate-400">根据温度趋势动态调整风扇偏移值</p>
            </div>
            <LedToggle :active="config.fanCurveOffsetEnabled" @click="handleOffsetToggle"
                       :disabled="!isConnected || !config.autoControl" />
          </div>
        </div>
        <div class="p-6 rounded-[2.5rem] border flex flex-col justify-between surface-card">
          <div>
            <h3 class="text-sm font-bold text-slate-700 dark:text-slate-300">温度采样次数</h3>
            <p class="text-[10px] text-slate-400 mt-1">设置响应灵敏度，次数越高越平滑</p>
          </div>
          <UnifiedSelect :model-value="effectiveSampleCount"
                         @update:model-value="handleSampleCountChange(Number($event))"
                         :options="sampleCountOptions"
                         :disabled="!config.autoControl"
                         :is-dark="isDark"
                         class="mt-4" />
        </div>
      </div>

      <!-- 运行模式与挡位 -->
      <div class="p-8 rounded-[2.5rem] border space-y-8 surface-card">
        <div>
          <h3 class="text-sm font-bold text-slate-700 dark:text-slate-300">运行模式与挡位</h3>
        </div>
        <div class="space-y-6">
          <div class="space-y-3">
            <label class="text-[10px] font-black text-slate-400 uppercase tracking-widest">预设模式</label>
            <div class="grid grid-cols-4 gap-3">
              <button v-for="mode in visiblePresetGears" :key="mode"
                      @click="handleGearChange(mode)"
                      @mousedown="startButtonAnim(`preset-${mode}`)"
                      :disabled="!isConnected || config.autoControl || config.customSpeedEnabled || gearTooHigh(mode)"
                      class="h-12 rounded-2xl font-bold text-xs transition-all flex items-center justify-center space-x-2 border cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                      :class="[
                        config.manualGear === mode
                          ? 'bg-blue-600 text-white border-blue-600 shadow-lg shadow-blue-200'
                          : 'surface-tile text-slate-600 dark:text-slate-300 border-slate-200/70 dark:border-white/10 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-200/60 dark:hover:bg-slate-700/30',
                        clickedButton === `preset-${mode}` ? 'mode-clicked' : ''
                      ]">
                <span>{{ mode }}</span>
              </button>
            </div>
          </div>
          <div class="space-y-3">
            <label class="text-[10px] font-black text-slate-400 uppercase tracking-widest">强度等级</label>
            <div class="flex items-center justify-between gap-4">
              <div class="flex bg-slate-100 dark:bg-white/[0.05] p-1 rounded-2xl w-fit">
                <button v-for="level in ['低', '中', '高']" :key="level"
                        @click="handleLevelChange(level)"
                        :disabled="!isConnected || config.autoControl || config.customSpeedEnabled || levelTooHigh(config.manualGear || '标准', level)"
                        class="px-8 py-2 rounded-xl text-xs font-bold transition-all disabled:opacity-40"
                        :class="config.manualLevel === level
                          ? 'bg-white dark:bg-slate-700 text-blue-600 dark:text-blue-400 shadow-sm'
                          : 'text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:hover:text-slate-300'">
                  {{ level }}
                </button>
              </div>
              <div class="flex items-center space-x-2">
                <span class="text-xs font-bold text-slate-700 dark:text-slate-300">最大支持挡位</span>
                <span class="text-xs font-black text-blue-600 dark:text-blue-400">{{ activeGearText }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

     <!-- 自定义转速 -->
     <div class="p-6 rounded-[2.5rem] border space-y-4 relative overflow-hidden surface-card">
       <div class="absolute -right-4 -bottom-4 w-32 h-32 rounded-full bg-blue-500/5 blur-[40px]" />
       <div class="flex justify-between items-start">
         <div>
           <h3 class="text-sm font-bold text-slate-700 dark:text-slate-300">自定义转速</h3>
           <p class="text-xs text-slate-400 mt-0.5">启用后将暂时关闭自动温控逻辑</p>
         </div>
         <LedToggle :active="config.customSpeedEnabled" @click="handleCustomSpeedToggle" :disabled="!isConnected" />
       </div>
        <div class="flex items-center space-x-3 relative z-10">
          <div class="flex-1 flex items-center border rounded-xl px-4 h-11 transition-all"
               :class="config.customSpeedEnabled
                 ? 'bg-slate-50 dark:bg-slate-800 border-blue-200 dark:border-blue-700'
                 : 'opacity-50 grayscale bg-slate-50 dark:bg-slate-800 border-slate-100 dark:border-slate-700'">
            <input type="text" v-model="customSpeedInput"
                   :disabled="!config.customSpeedEnabled"
                   inputmode="numeric"
                   class="bg-transparent border-none outline-none w-full text-sm font-bold text-slate-700 dark:text-slate-300" />
            <span class="text-[10px] font-black text-slate-400 dark:text-slate-500">RPM</span>
          </div>
          <button @click="handleCustomSpeedApply" @mousedown="startButtonAnim('custom-apply')"
                  :disabled="!config.customSpeedEnabled || !isConnected || loadingCustomSpeed || !isCustomSpeedInputValid || customSpeedAboveLimit"
                  class="h-11 px-6 rounded-xl font-bold text-xs transition-all cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                  :class="[
                    config.customSpeedEnabled && isConnected
                      ? 'bg-blue-600 text-white shadow-lg shadow-blue-200 hover:bg-blue-700 hover:shadow-blue-300/60'
                      : 'bg-slate-100 dark:bg-slate-800 text-slate-300 dark:text-slate-600',
                    clickedButton === 'custom-apply' ? 'mode-clicked' : ''
                  ]">
            {{ loadingCustomSpeed ? '应用中...' : '应用' }}
          </button>
        </div>
        <!-- 输入范围提示 -->
        <p v-if="config.customSpeedEnabled" class="text-[10px] text-slate-400">
          有效范围：500 – 4500 RPM，按 100 取整
          <span v-if="customSpeedInput && !isCustomSpeedInputValid" class="text-rose-500 ml-2">当前输入不可应用</span>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import LedToggle from '../components/ui/LedToggle.vue'
import UnifiedSelect from '../components/ui/UnifiedSelect.vue'
import { apiService } from '../services/api'
import { frontendLogger } from '../services/frontendLogger'
import { types } from '../../wailsjs/go/models'

interface Props {
  isDark: boolean
  isConnected: boolean
  deviceModel?: string
  deviceProductId?: string
  config: types.AppConfig
  fanData: types.FanData | null
}
const props = defineProps<Props>()
const emit = defineEmits<{ 'config-change': [config: types.AppConfig] }>()

const gearOrder = ['静音', '标准', '强劲', '超频']
const levelOrder = ['低', '中', '高']

function isPlainModel(model: string | undefined, productId: string | undefined) {
  const pid = (productId || '').toLowerCase()
  if (pid.includes('1001')) return true
  if (pid.includes('1002')) return false
  return (model || '').toUpperCase().replace(/\s+/g, '') === 'BS2'
}

function gearSpeedsFor(plain: boolean): Record<string, number[]> {
  return {
    '静音': [1300, 1700, 1900],
    '标准': [2100, 2400, 2700],
    '强劲': plain ? [2800, 2900, 3000] : [2800, 3000, 3300],
    ...(plain ? {} : { '超频': [3500, 3700, 4000] }),
  }
}

function rpmLimitForModel(model: string | undefined, productId: string | undefined, supply: number | undefined) {
  const pid = (productId || '').toLowerCase()
  let plain = false
  if (pid.includes('1001')) plain = true
  else if (!pid.includes('1002')) {
    plain = (model || '').toUpperCase().replace(/\s+/g, '') === 'BS2'
  }
  if (supply === 1) return 2700
  if (plain) return 3000
  if (supply === 2) return 3300
  return 4000
}

const speedLimit = computed(() => rpmLimitForModel(props.deviceModel, props.deviceProductId, props.fanData?.vbus))
const gearSpeeds = computed(() => gearSpeedsFor(isPlainModel(props.deviceModel, props.deviceProductId)))
const visiblePresetGears = computed(() => gearOrder.filter(gear => (gearSpeeds.value[gear] || []).some(rpm => rpm <= speedLimit.value)))

function levelRpm(gear: string, level: string) {
  const speeds = gearSpeeds.value[gear]
  const index = levelOrder.indexOf(level)
  if (!speeds || index < 0) return Number.POSITIVE_INFINITY
  return speeds[index]
}

function levelTooHigh(gear: string, level: string) {
  return levelRpm(gear, level) > speedLimit.value
}

function gearTooHigh(gear: string) {
  return levelOrder.every(level => levelTooHigh(gear, level))
}

function highestUsableLevel(gear: string) {
  for (let i = levelOrder.length - 1; i >= 0; i--) {
    const level = levelOrder[i]
    if (!levelTooHigh(gear, level)) return level
  }
  return ''
}

const activeGearText = computed(() => {
  if (!props.isConnected) return '—'
  for (let i = visiblePresetGears.value.length - 1; i >= 0; i--) {
    const gear = visiblePresetGears.value[i]
    const speeds = (gearSpeeds.value[gear] || []).filter(rpm => rpm <= speedLimit.value)
    if (speeds.length > 0) return `${gear} ${speeds[speeds.length - 1]}转`
  }
  return '—'
})

const customSpeedInput = ref(String(props.config.customSpeedRPM || 2000))
const loadingCustomSpeed = ref(false)
const clickedButton = ref<string | null>(null)
let buttonAnimTimer: ReturnType<typeof setTimeout> | null = null

watch(() => props.config.customSpeedRPM, (v) => {
  if (v !== undefined && v !== null) customSpeedInput.value = String(v)
})

function startButtonAnim(key: string) {
  clickedButton.value = key
  if (buttonAnimTimer) clearTimeout(buttonAnimTimer)
  buttonAnimTimer = setTimeout(() => { clickedButton.value = null }, 200)
}

onUnmounted(() => {
  if (buttonAnimTimer) clearTimeout(buttonAnimTimer)
})

const parsedCustomSpeed = computed(() => {
  const v = Number(customSpeedInput.value)
  return Number.isFinite(v) ? v : NaN
})
const isCustomSpeedInputValid = computed(() =>
  Number.isFinite(parsedCustomSpeed.value) && parsedCustomSpeed.value >= 500 && parsedCustomSpeed.value <= 4500
)
const customSpeedAboveLimit = computed(() =>
  Number.isFinite(parsedCustomSpeed.value) && (
    parsedCustomSpeed.value > speedLimit.value || normalizeCustomSpeed(parsedCustomSpeed.value) > speedLimit.value
  )
)

function normalizeCustomSpeed(rpm: number): number {
  const step = 100
  let value = Math.round(rpm / step) * step
  if (value < 500) value = 500
  if (value > 4500) value = 4500
  return value
}

const allSampleOptions = [
  { value: 1, label: '1次 (即时响应)' }, { value: 2, label: '2次 (2秒平均)' },
  { value: 3, label: '3次 (3秒平均)' }, { value: 5, label: '5次 (5秒平均)' },
  { value: 10, label: '10次 (10秒平均)' },
]
const sampleCountOptions = computed(() =>
  props.config.fanCurveOffsetEnabled ? allSampleOptions.filter(o => o.value >= 3) : allSampleOptions
)
const effectiveSampleCount = computed(() =>
  props.config.fanCurveOffsetEnabled ? Math.max(props.config.tempSampleCount || 1, 3) : (props.config.tempSampleCount || 1)
)

async function handleAutoControlToggle() {
  if (!props.isConnected) return
  try {
    const nextAuto = !props.config.autoControl
    if (nextAuto && props.config.customSpeedEnabled) {
      await apiService.setCustomSpeed(false, props.config.customSpeedRPM || 2000)
    }
    await apiService.setAutoControl(nextAuto)
    emit('config-change', types.AppConfig.createFrom({
      ...props.config,
      autoControl: nextAuto,
      customSpeedEnabled: nextAuto ? false : props.config.customSpeedEnabled,
    }))
  } catch (e) { frontendLogger.error('设备参数', '切换自动温控失败', e) }
}

async function handleOffsetToggle() {
  if (!props.isConnected || !props.config.autoControl) return
  const enabled = !props.config.fanCurveOffsetEnabled
  try {
    const safeSample = enabled && (props.config.tempSampleCount || 1) < 3 ? 3 : (props.config.tempSampleCount || 1)
    const nc = types.AppConfig.createFrom({ ...props.config, fanCurveOffsetEnabled: enabled, tempSampleCount: safeSample })
    await apiService.updateConfig(nc)
    emit('config-change', nc)
  } catch (e) { frontendLogger.error('设备参数', '切换自动曲线偏移失败', e) }
}

async function handleSampleCountChange(count: number) {
  try {
    const nc = types.AppConfig.createFrom({ ...props.config, tempSampleCount: count })
    await apiService.updateConfig(nc)
    emit('config-change', nc)
  } catch (e) { frontendLogger.error('设备参数', '修改温度采样次数失败', e) }
}

async function handleCustomSpeedToggle() {
  if (!props.isConnected) return
  const enabled = !props.config.customSpeedEnabled
  const rpm = isCustomSpeedInputValid.value ? parsedCustomSpeed.value : (props.config.customSpeedRPM || 2000)
  await handleCustomSpeedApplyWith(enabled, rpm)
}

async function handleCustomSpeedApply() {
  if (!props.config.customSpeedEnabled || !props.isConnected) return
  if (!isCustomSpeedInputValid.value || customSpeedAboveLimit.value) return
  await handleCustomSpeedApplyWith(true, parsedCustomSpeed.value)
}

async function handleCustomSpeedApplyWith(enabled: boolean, rpm: number) {
  rpm = normalizeCustomSpeed(rpm)
  if (enabled && rpm > speedLimit.value) return
  customSpeedInput.value = String(rpm)
  loadingCustomSpeed.value = true
  try {
    await apiService.setCustomSpeed(enabled, rpm)
    emit('config-change', types.AppConfig.createFrom({
      ...props.config, customSpeedEnabled: enabled, customSpeedRPM: rpm,
      autoControl: enabled ? false : props.config.autoControl,
    }))
  } catch (e) { frontendLogger.error('设备参数', '设置自定义转速失败', e) }
  finally { loadingCustomSpeed.value = false }
}

async function handleGearChange(gear: string) {
  if (!props.isConnected || props.config.autoControl || props.config.customSpeedEnabled) return
  const level = highestUsableLevel(gear)
  if (!level) return
  try {
    await apiService.setManualGear(gear, level)
    emit('config-change', types.AppConfig.createFrom({ ...props.config, manualGear: gear, manualLevel: level }))
  } catch (e) { frontendLogger.error('设备参数', '切换预设模式失败', e) }
}

async function handleLevelChange(level: string) {
  const gear = props.config.manualGear || '标准'
  if (levelTooHigh(gear, level)) return
  if (!props.isConnected || props.config.autoControl || props.config.customSpeedEnabled) return
  try {
    await apiService.setManualGear(props.config.manualGear || '标准', level)
    emit('config-change', types.AppConfig.createFrom({ ...props.config, manualLevel: level }))
  } catch (e) { frontendLogger.error('设备参数', '切换强度等级失败', e) }
}

</script>
