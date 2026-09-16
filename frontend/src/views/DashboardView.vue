<template>
  <div class="dashboard-home relative flex flex-col flex-1 min-h-0 overflow-hidden" :class="{ offline: !isConnected }">
    <div class="dash-fog" aria-hidden="true">
      <i class="dash-fog-core" />
      <i class="dash-fog-drift" />
    </div>

    <header class="relative z-10 flex items-center justify-between px-9 pt-6">
      <h2 class="text-xl font-bold tracking-tight text-slate-800 dark:text-white">设备概览</h2>
      <div class="dash-mode" :class="{ 'is-offline': !isConnected }">
        <button type="button" :class="{ on: isSmartFreq }" :disabled="!isConnected" @click="setMode(true)">自动</button>
        <button type="button" :class="{ on: !isSmartFreq }" :disabled="!isConnected" @click="setMode(false)">手动</button>
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
const emit = defineEmits<{ 'update:isSmartFreq': [v: boolean] }>()

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
  return `飞智 ${sku}`
})

const lastTargetRpm = ref<number | null>(null)

watch(() => props.fanData?.targetRpm, (rpm) => {
  if (rpm && rpm > 0) lastTargetRpm.value = rpm
}, { immediate: true })

watch(() => props.isSmartFreq, (enabled) => {
  if (!enabled) lastTargetRpm.value = null
})

const currentGear = computed(() => props.fanData?.setGear || '')

const fanMaxRpm = computed(() => {
  const t = props.fanData?.maxGear || ''
  if (t.includes('超频')) return 4000
  if (t.includes('强劲')) return 3300
  return 2760
})

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
  const gear = currentGear.value || props.config.manualGear || '—'
  if (gear.includes('超频')) return ['超频']
  const level = props.config.manualLevel || '中'
  return [gear, level]
})

function setMode(auto: boolean) {
  if (!props.isConnected || auto === props.isSmartFreq) return
  emit('update:isSmartFreq', auto)
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
  right: -80px;
  bottom: -120px;
  width: 860px;
  height: 560px;
  pointer-events: none;
  z-index: 0;
}
.dark .dash-fog {
  --dash-mist: rgb(52 211 153 / 0.12);
  --dash-mist-soft: rgb(110 231 183 / 0.05);
}
.dash-fog-core,
.dash-fog-drift {
  position: absolute;
  inset: 0;
  filter: blur(56px);
}
.dash-fog-core {
  background: radial-gradient(ellipse 88% 76% at 62% 68%, var(--dash-mist) 0%, transparent 78%);
  animation: dash-fog-breathe 18s ease-in-out infinite;
}
.dash-fog-drift {
  background: radial-gradient(ellipse 96% 84% at 48% 72%, var(--dash-mist-soft) 0%, transparent 82%);
  animation: dash-fog-shift 26s ease-in-out infinite;
}
.dashboard-home.offline .dash-fog-core,
.dashboard-home.offline .dash-fog-drift {
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
  0%, 100% { transform: translate(0, 0) scale(1, 1); }
  50% { transform: translate(10px, -6px) scale(1.03, 0.98); }
}
@keyframes dash-fog-shift {
  0%, 100% { transform: translate(0, 0) scale(1.01, 1); }
  50% { transform: translate(-12px, 6px) scale(0.98, 1.03); }
}
@media (prefers-reduced-motion: reduce) {
  .dash-fog-core,
  .dash-fog-drift { animation: none; }
  .dash-track > i { transition: none; }
}
</style>
