import {readable} from 'svelte/store'
import {GameNet} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import type {gamenet} from '../../wailsjs/go/models'
import type {Key} from './i18n'
import {worst, type Level} from './net'

export type GameNetStatus = gamenet.Status
export type Flow = gamenet.Flow

// Latest report on the game's connections from the measuring helper; null
// while the helper is not running. In the overlay the events arrive over IPC.
export function gameNet() {
  return readable<GameNetStatus | null>(null, (set) => {
    GameNet()
      .then((s) => set(s.state ? s : null))
      .catch(() => {})
    const offStats = EventsOn('game:net', (s: GameNetStatus) => set(s))
    const offState = EventsOn('fps:state', (on: boolean) => {
      if (!on) set(null)
    })
    return () => {
      offStats()
      offState()
    }
  })
}

const kindLabel: Record<string, Key> = {
  relay: 'gnet.relay',
  peer: 'gnet.peer',
  server: 'gnet.server',
  lan: 'gnet.lan',
}
export const kind = (k: string): Key => kindLabel[k] ?? 'gnet.peer'

// A silence this long from the other side is felt in game (rubber-banding,
// late hits). Heuristic, not taken from the game.
export const gapWarnMs = 400
export const gapBadMs = 1000

export function gapLevel(f: Flow): Level {
  if (f.proto !== 'udp' || f.pktsInPerSec === 0) return 'unknown' // silence is normal on TCP
  if (f.maxGapMs >= gapBadMs) return 'bad'
  if (f.maxGapMs >= gapWarnMs) return 'warn'
  return 'good'
}

// Many peers and relays drop ICMP; after a few unanswered probes we say so
// instead of reporting 100% loss.
export const pingNoReply = (f: Flow) => !!f.ping && f.ping.sent >= 5 && f.ping.received === 0
export const hasPing = (f: Flow) => !!f.ping && f.ping.received > 0

export function flowLevel(f: Flow): Level {
  const levels: string[] = [gapLevel(f)]
  if (f.ping && hasPing(f)) levels.push(f.ping.level)
  return worst(levels)
}

// The connection that matters most: the busiest UDP flow still active.
export function mainFlow(s: GameNetStatus | null): Flow | null {
  return s?.flows?.find((f) => f.proto === 'udp' && f.idleMs < 5000) ?? null
}
