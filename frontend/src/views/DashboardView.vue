<template>
  <div class="dashboard-home relative flex flex-col flex-1 min-h-0 overflow-hidden" :class="{ offline: !isConnected }">
    <div class="dash-fog" aria-hidden="true">
      <i class="dash-fog-core" />
      <i class="dash-fog-drift" />
    </div>

    <header class="relative z-20 flex items-start justify-between px-8 pt-8">
      <h2 class="text-xl font-bold leading-7 tracking-tight text-slate-800 dark:text-white">设备概览</h2>
      <div class="relative">
        <div class="dash-mode" :class="{ 'is-offline': !isConnected }">
          <button type="button" :class="{ on: isSmartFreq }" :disabled="!isConnected" @click="setMode(true)">自动</button>
          <button type="button" :class="{ on: !isSmartFreq }" :disabled="!isConnected" @click="setMode(false)">手动</button>
        </div>
        <div v-if="!isSmartFreq" class="dash-mode absolute top-full right-0 mt-2" :class="{ 'is-offline': !isConnected || config.customSpeedEnabled }">
          <button v-for="(gear, index) in basicGears" :key="gear" type="button"
                  :class="{ on: config.manualGear === gear, blocked: index >= gearCutoff }"
                  :disabled="(!isConnected || config.customSpeedEnabled) && index < gearCutoff"
                  @click="setGear(gear, index)">{{ gear }}</button>
        </div>
      </div>
    </header>
    <div class="dash-model relative z-10 px-9 pt-6">{{ wordmark }}</div>

    <section class="dash-instrument relative z-10">
      <div class="dash-hero">
        <div class="tabular-nums">
          <span class="dash-rpm">{{ rpmText }}</span>
          <span class="dash-rpm-unit">RPM</span>
        </div>
        <div class="dash-meta">
          <template v-for="(part, i) in metaParts" :key="i">
            <span v-if="i > 0" class="dash-sep">/</span>
            <b v-if="i === 0">{{ part }}</b>
            <span v-else>{{ part }}</span>
          </template>
        </div>
        <div class="dash-track"><i :style="{ width: fanProgress + '%' }" /></div>
      </div>
      <div class="dash-split" />
      <div class="dash-temps">
        <div class="dash-temp">
          <div class="dash-temp-label">CPU</div>
          <div class="tabular-nums">
            <span class="dash-temp-num" :class="cpuTone.cls">{{ cpuText }}</span>
            <span class="dash-temp-unit">°C</span>
          </div>
          <div class="dash-temp-state" :class="cpuTone.cls">{{ cpuTone.label }}</div>
        </div>
        <div class="dash-temp">
          <div class="dash-temp-label">GPU</div>
          <div class="tabular-nums">
            <span class="dash-temp-num" :class="gpuTone.cls">{{ gpuText }}</span>
            <span class="dash-temp-unit">°C</span>
          </div>
          <div class="dash-temp-state" :class="gpuTone.cls">{{ gpuTone.label }}</div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { types } from '../../wailsjs/go/models'
import { apiService } from '../services/api'
import { frontendLogger } from '../services/frontendLogger'

interface Props {
  isConnected: boolean
  deviceModel?: string
  deviceProductId?: string
  isSmartFreq: boolean
  fanData: types.FanData | null
  temperature: types.TemperatureData | null
  config: types.AppConfig
}
const props = defineProps<Props>()
const emit = defineEmits<{ 'update:isSmartFreq': [v: boolean]; 'config-change': [config: types.AppConfig] }>()
const basicGears = ['静音', '标准', '强劲', '超频']

function unavailableGearIndex(model: string | undefined, productId: string | undefined, supply: number | undefined) {
  const pid = productId || ''
  const plain = pid.includes('1001') || (!pid.includes('1002') && (model || '').replace(/ /g, '') === 'BS2')
  if (plain) return supply === 3 || supply === 2 ? 3 : 2
  if (supply === 3) return 4
  if (supply === 2) return 3
  return 2
}

const gearCutoff = computed(() => unavailableGearIndex(props.deviceModel, props.deviceProductId, props.fanData?.vbus))

const wordmark = computed(() => {
  const pid = (props.deviceProductId || '').toLowerCase()
  let sku = 'BS2 PRO'
  if (pid.includes('1001')) sku = 'BS2'
  else if (pid.includes('1002')) sku = 'BS2 PRO'
  else {
    const model = (props.deviceModel || '').toUpperCase().replace(/\s+/g, '')
    if (model.includes('BS2PRO')) sku = 'BS2 PRO'
    else if (model.includes('BS2')) sku = 'BS2'
  }
  return sku
})

const lastTargetRpm = ref<number | null>(null)

watch(() => props.fanData?.targetRpm, (rpm) => {
  if (rpm && rpm > 0) lastTargetRpm.value = rpm
}, { immediate: true })

watch(() => props.isSmartFreq, (enabled) => {
  if (!enabled) lastTargetRpm.value = null
})

function rpmLimitForModel(model: string | undefined, productId: string | undefined, vbus: number | undefined) {
  const pid = (productId || '').toLowerCase()
  let plain = false
  if (pid.includes('1001')) plain = true
  else if (!pid.includes('1002')) {
    plain = (model || '').toUpperCase().replace(/\s+/g, '') === 'BS2'
  }
  if (vbus === 1) return 2700
  if (plain) return 3000
  if (vbus === 2) return 3300
  return 4000
}

const fanMaxRpm = computed(() => rpmLimitForModel(props.deviceModel, props.deviceProductId, props.fanData?.vbus))

const rpmText = computed(() => props.isConnected ? String(props.fanData?.currentRpm ?? '—') : '—')
const cpuText = computed(() => props.isConnected ? String(props.temperature?.cpuTemp ?? '—') : '—')
const gpuText = computed(() => props.isConnected ? String(props.temperature?.gpuTemp ?? '—') : '—')
const fanProgress = computed(() => {
  if (!props.isConnected) return 0
  return Math.min(100, ((props.fanData?.currentRpm ?? 0) / fanMaxRpm.value) * 100)
})

function tempTone(temp: number | null) {
  if (temp == null) return { cls: 'off', label: '' }
  if (temp > 85) return { cls: 'crit', label: '过热' }
  if (temp > 75) return { cls: 'high', label: '偏高' }
  if (temp > 60) return { cls: 'norm', label: '正常' }
  return { cls: 'ok', label: '良好' }
}

const cpuTone = computed(() => props.isConnected ? tempTone(props.temperature?.cpuTemp ?? null) : tempTone(null))
const gpuTone = computed(() => props.isConnected ? tempTone(props.temperature?.gpuTemp ?? null) : tempTone(null))

const metaParts = computed(() => {
  if (!props.isConnected) return ['等待设备']
  if (props.config.customSpeedEnabled) return [`自定义 ${props.config.customSpeedRPM}`]
  if (props.isSmartFreq) {
    const target = props.fanData?.targetRpm || lastTargetRpm.value
    const parts = [`目标 ${target || '—'}`, '自动']
    if (props.config.fanCurveOffsetEnabled && props.temperature) {
      const off = props.temperature.autoOffset ?? 0
      parts.push(`${off >= 0 ? '+' : ''}${off} RPM`)
      if (props.temperature.engineState) parts.push(props.temperature.engineState)
    }
    return parts
  }
  const gear = props.config.manualGear || '—'
  if (gear.includes('超频')) return ['超频']
  const level = props.config.manualLevel || '中'
  return [gear, level]
})

function setMode(auto: boolean) {
  if (!props.isConnected || auto === props.isSmartFreq) return
  emit('update:isSmartFreq', auto)
}

async function setGear(gear: string, index: number) {
  if (index >= gearCutoff.value) return
  if (!props.isConnected || props.isSmartFreq || props.config.customSpeedEnabled || gear === props.config.manualGear) return
  try {
    await apiService.setManualGear(gear, props.config.manualLevel || '中')
    emit('config-change', types.AppConfig.createFrom({ ...props.config, manualGear: gear }))
  } catch (e) { frontendLogger.error('设备概览', '切换预设模式失败', e) }
}
</script>

<style>
.dashboard-home.offline .dash-fog {
  --dash-mist: rgb(148 163 184 / 0.08);
  --dash-mist-soft: rgb(148 163 184 / 0.04);
}
.dash-fog {
  --dash-mist: rgb(16 185 129 / 0.10);
  --dash-mist-soft: rgb(167 243 208 / 0.05);
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  z-index: 0;
}
.dark .dash-fog {
  --dash-mist: rgb(52 211 153 / 0.12);
  --dash-mist-soft: rgb(110 231 183 / 0.05);
}
.dash-fog i {
  position: absolute;
  left: 0;
  top: 0;
  filter: blur(64px);
}
.dash-fog-core {
  width: 64%;
  height: 76%;
  border-radius: 46% 54% 48% 52%;
  background: radial-gradient(ellipse at 50% 50%, var(--dash-mist) 0%, transparent 68%);
  animation: dash-fog-breathe 52s ease-in-out infinite;
}
.dash-fog-drift {
  width: 58%;
  height: 68%;
  border-radius: 58% 42% 52% 48%;
  background: radial-gradient(ellipse at 50% 50%, var(--dash-mist-soft) 0%, transparent 70%);
  animation: dash-fog-shift 68s ease-in-out infinite;
}
.dashboard-home.offline .dash-fog i {
  animation-play-state: paused;
}
.dash-mode {
  display: flex;
  padding: 4px;
  border-radius: 16px;
  gap: 2px;
  background: rgba(241, 245, 249, 0.92);
  box-shadow: inset 0 0 0 1px rgba(203, 213, 225, 0.7);
}
.dark .dash-mode {
  background: rgba(255,255,255,0.05);
  box-shadow: inset 0 0 0 1px rgba(255,255,255,0.08);
}
.dash-mode button {
  border: 0;
  background: transparent;
  min-width: 64px;
  height: 32px;
  padding: 0 14px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  color: #94a3b8;
}
.dash-mode button.blocked {
  opacity: 0.4;
  cursor: not-allowed;
  pointer-events: auto;
}
.dash-mode button.on {
  background: #fff;
  color: #2563eb;
  box-shadow: 0 1px 4px rgba(15,23,42,0.08);
}
.dark .dash-mode button.on {
  background: rgba(51, 65, 85, 0.9);
  color: #60a5fa;
}
.dash-mode.is-offline {
  opacity: 0.4;
  pointer-events: none;
}
.dash-model {
  font-size: 44px;
  font-weight: 800;
  letter-spacing: -0.05em;
  line-height: 1;
  color: #0f172a;
}
.dark .dash-model { color: #f8fafc; }
.dashboard-home.offline .dash-model { color: #94a3b8; }
.dash-instrument {
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) 1px minmax(0, 0.85fr);
  gap: 0 28px;
  align-items: center;
  min-height: 0;
  padding: 0 36px 36px;
}
.dash-split {
  align-self: stretch;
  background: linear-gradient(to bottom, transparent, rgba(148,163,184,0.35), transparent);
}
.dark .dash-split {
  background: linear-gradient(to bottom, transparent, rgba(255,255,255,0.08), transparent);
}
.dash-rpm {
  font-size: 84px;
  font-weight: 800;
  letter-spacing: -0.06em;
  line-height: 0.92;
  color: #2563eb;
}
.dark .dash-rpm { color: #60a5fa; }
.dashboard-home.offline .dash-rpm { color: #94a3b8; }
.dash-rpm-unit {
  margin-left: 10px;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: #64748b;
  vertical-align: 18px;
}
.dash-meta {
  margin-top: 14px;
  font-size: 13px;
  color: #64748b;
  min-height: 20px;
}
.dash-meta b { font-weight: 650; color: #475569; }
.dark .dash-meta b { color: #cbd5e1; }
.dash-sep { margin: 0 10px; color: #cbd5e1; font-weight: 300; }
.dark .dash-sep { color: #475569; }
.dash-track {
  margin-top: 22px;
  height: 2px;
  width: 100%;
  max-width: 320px;
  background: rgba(148,163,184,0.22);
  border-radius: 99px;
  overflow: hidden;
}
.dash-track > i {
  display: block;
  height: 100%;
  background: #2563eb;
  border-radius: 99px;
  transition: width 700ms ease;
}
.dash-temps {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 32px;
}
.dash-temp-label {
  font-size: 12px;
  font-weight: 650;
  color: #64748b;
  margin-bottom: 6px;
}
.dash-temp-num {
  font-size: 40px;
  font-weight: 800;
  letter-spacing: -0.04em;
  line-height: 1;
}
.dash-temp-unit {
  font-size: 12px;
  font-weight: 700;
  margin-left: 4px;
  color: #94a3b8;
}
.dash-temp-state { margin-top: 8px; font-size: 12px; }
.dash-temp-num.ok, .dash-temp-state.ok { color: #059669; }
.dash-temp-num.norm, .dash-temp-state.norm { color: #ca8a04; }
.dash-temp-num.high, .dash-temp-state.high { color: #ea580c; }
.dash-temp-num.crit, .dash-temp-state.crit { color: #ef4444; }
.dash-temp-num.off, .dash-temp-state.off { color: #94a3b8; }
.dark .dash-temp-num.ok, .dark .dash-temp-state.ok { color: #34d399; }
.dark .dash-temp-num.norm, .dark .dash-temp-state.norm { color: #facc15; }
@keyframes dash-fog-breathe {
  0%, 100% { transform: translate(-8%, 36%) scale(1.04, 0.94); opacity: 0.88; }
  25% { transform: translate(60%, -12%) scale(0.92, 1.08); opacity: 1; }
  50% { transform: translate(-10%, -12%) scale(1.08, 0.9); opacity: 0.78; }
  75% { transform: translate(58%, 36%) scale(0.96, 1.04); opacity: 0.95; }
}
@keyframes dash-fog-shift {
  0%, 100% { transform: translate(78%, -10%) scale(1.02, 1); opacity: 0.72; }
  32% { transform: translate(-12%, 52%) scale(0.9, 1.12); opacity: 1; }
  64% { transform: translate(76%, 52%) scale(1.08, 0.94); opacity: 0.7; }
}
@media (prefers-reduced-motion: reduce) {
  .dash-fog i { animation: none; }
  .dash-track > i { transition: none; }
}
</style>
