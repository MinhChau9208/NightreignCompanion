import {readable} from 'svelte/store'
import {TimerState} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import type {timer} from '../../wailsjs/go/models'
import type {Key} from './i18n'

export type Snapshot = timer.Snapshot

// Latest timer state from the main process. The overlay cannot call
// TimerState, so it waits for the next "timer" event (at most a second).
export function timerState(fetch = true) {
  return readable<Snapshot | null>(null, (set) => {
    if (fetch) TimerState().then(set).catch(() => {})
    return EventsOn('timer', set)
  })
}

// mm:ss, or h:mm:ss past an hour.
export function clock(sec: number) {
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  const mm = String(m).padStart(2, '0')
  const ss = String(s).padStart(2, '0')
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`
}

export const kindKey = (kind: string) => `timer.kind.${kind}` as Key
export const stateKey = (state: string) => `timer.state.${state}` as Key

// The big number: time left in a timed phase, fight time in a boss phase.
export function mainClock(s: Snapshot) {
  if (s.state === 'done') return clock(s.elapsed)
  if (s.state === 'idle') return '--:--'
  return clock(s.duration > 0 ? s.remaining : s.phaseElapsed)
}

// Warn as the end of a timed phase gets close.
export function urgency(s: Snapshot | null) {
  if (!s || s.duration === 0 || (s.state !== 'running' && s.state !== 'paused')) return ''
  if (s.remaining <= 10) return 'bad'
  if (s.remaining <= 60) return 'warn'
  return ''
}
