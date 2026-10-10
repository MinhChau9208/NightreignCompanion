import {readable} from 'svelte/store'
import {FPSRunning} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import type {Level} from './net'

// Mirrors internal/fps.Status. Declared by hand because the Go struct
// embeds FrameStats, which the Wails model generator does not flatten.
export type FPSStatus = {
  state: 'starting' | 'waiting' | 'running' | 'error'
  process: string
  pid: number
  fps: number
  avgFps: number
  low1Fps: number
  frametimeMs: number
  maxFrametimeMs: number
  frames: number
  history: number[] | null
  error?: string
}

// Latest helper report; cleared when the helper exits, except that an
// error report is kept so the user can still read why it stopped.
export function fpsStatus() {
  return readable<FPSStatus | null>(null, (set) => {
    let last: FPSStatus | null = null
    const offStats = EventsOn('fps:stats', (s: FPSStatus) => set((last = s)))
    const offState = EventsOn('fps:state', (on: boolean) => {
      if (!on && last?.state !== 'error') set((last = null))
    })
    return () => {
      offStats()
      offState()
    }
  })
}

export function fpsRunning() {
  return readable(false, (set) => {
    FPSRunning().then(set).catch(() => {})
    return EventsOn('fps:state', (on: boolean) => set(on))
  })
}

// Bad when the current FPS is under the minimum; warning when only the
// 1% lows dip under it (stutter).
export function fpsLevel(s: FPSStatus | null, minFps: number): Level {
  if (!s || s.state !== 'running' || s.frames < 2) return 'unknown'
  if (s.fps < minFps) return 'bad'
  if (s.low1Fps < minFps) return 'warn'
  return 'good'
}
