export const ACTIVE_PROFILE_NAME_KEY = 'bs2pro.fanCurve.activeProfileName'

export interface ProfileNameRef {
  id: string
  name: string
}

export interface KeyValueStore {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}

export function readActiveProfileName(store: KeyValueStore): string {
  try {
    return (store.getItem(ACTIVE_PROFILE_NAME_KEY) || '').trim()
  } catch {
    return ''
  }
}

export function writeActiveProfileName(store: KeyValueStore, name: string) {
  try {
    store.setItem(ACTIVE_PROFILE_NAME_KEY, name.trim())
  } catch {
    // implement: localStorage only, profile json if another client must share the selection
  }
}

export function resolveSelectedProfileId(profiles: ProfileNameRef[], storedName: string, deviceProfileId: string): string {
  const match = profiles.find(profile => profile.name === storedName.trim())
  return match?.id || deviceProfileId
}

