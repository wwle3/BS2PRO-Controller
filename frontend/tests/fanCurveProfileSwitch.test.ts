import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import {
  ACTIVE_PROFILE_NAME_KEY,
  readActiveProfileName,
  resolveSelectedProfileId,
  writeActiveProfileName,
} from '../src/services/fanCurveProfileSwitch.ts'

const memory = () => {
  const data = new Map<string, string>()
  return {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => { data.set(key, value) },
  }
}

const store = memory()
assert.equal(readActiveProfileName(store), '')
writeActiveProfileName(store, '配置 1')
assert.equal(store.getItem(ACTIVE_PROFILE_NAME_KEY), '配置 1')
assert.equal(readActiveProfileName(store), '配置 1')

const profiles = [
  { id: 'device-default', name: '默认配置' },
  { id: 'profile-1', name: '配置 1' },
]
assert.equal(resolveSelectedProfileId(profiles, '配置 1', 'device-default'), 'profile-1')
assert.equal(resolveSelectedProfileId(profiles, '不存在', 'device-default'), 'device-default')
assert.equal(resolveSelectedProfileId(profiles, '', 'device-default'), 'device-default')

const vue = readFileSync(new URL('../src/views/FanCurveView.vue', import.meta.url), 'utf8').replace(/\r\n/g, '\n')
const init = vue.slice(vue.indexOf('async function initProfiles'), vue.indexOf('function updateSelectedProfileCurve'))
assert.equal(init.includes('resolveSelectedProfileId'), true, '回到曲线页应按配置名恢复正在生效的配置')
assert.equal(init.includes('await refreshVisibleCurve()'), true, '配置放好后再取正在生效的偏移')
const apply = vue.slice(vue.indexOf('function applySelectedCurve'), vue.indexOf('function switchProfile'))
assert.equal(apply.includes('writeActiveProfileName'), true)
const switched = vue.slice(vue.indexOf('function switchProfile'), vue.indexOf('function createProfile'))
assert.equal(switched.includes('nextId === selectedProfileId.value) return'), true, '再点当前配置不能重新下发')
assert.equal(switched.includes('applySelectedCurve()'), true)
const created = vue.slice(vue.indexOf('function createProfile'), vue.indexOf('function deleteProfile'))
assert.equal(created.includes('applySelectedCurve()'), true)
const deleted = vue.slice(vue.indexOf('function deleteProfile'), vue.indexOf('const selectedProfile ='))
assert.equal(deleted.includes('applySelectedCurve()'), true)
const renamed = vue.slice(vue.indexOf('function submitRenameProfile'), vue.indexOf('function cancelRenameProfile'))
assert.equal(renamed.includes('writeActiveProfileName'), true)
assert.equal(vue.includes('function shownOffset'), true, '黄线读可见曲线，不写进草稿')
assert.equal(vue.includes('paintVisibleOffsets'), false)
const reset = vue.slice(vue.indexOf('function resetCurve'), vue.indexOf('function exportConfig'))
assert.equal(reset.includes('resetFanOffset'), false, '重置默认只改草稿，不能清掉正在生效的偏移')
assert.equal(reset.includes('resetClearsOffset.value = true'), true, '重置保存时要清掉这条曲线的旧偏移')
const save = vue.slice(vue.indexOf('async function saveCurve'), vue.indexOf('async function applyOffsetCurve'))
assert.equal(save.includes('resetFanCurve'), true, '保存重置结果时删除长期记忆')
assert.equal(vue.slice(vue.indexOf('function createProfile'), vue.indexOf('function deleteProfile')).includes('resetClearsOffset.value = false'), true)
assert.equal(vue.slice(vue.indexOf('function deleteProfile'), vue.indexOf('async function renameProfile')).includes('resetClearsOffset.value = false'), true)
const cancel = vue.slice(vue.indexOf('function cancelChanges'), vue.indexOf('function resetCurve'))
assert.equal(cancel.includes('refreshVisibleCurve()'), true, '取消后要把原来的黄线读回来')

console.log('fan curve profile switch: ok')
