import {readable} from 'svelte/store'
import {NetStats} from '../../wailsjs/go/main/App'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import type {netmon} from '../../wailsjs/go/models'
import type {Key} from './i18n'

export type NetStat = netmon.Stats
export type Level = 'good' | 'warn' | 'bad' | 'unknown'

// Live network stats, pushed by the backend once per second. In the overlay
// process NetStats is empty and the events arrive relayed over IPC.
export function netStats() {
  return readable<NetStat[]>([], (set) => {
    NetStats().then(set).catch(() => {})
    return EventsOn('net:stats', (s: NetStat[]) => set(s))
  })
}

// Status is never color-alone: every level has an icon and a label.
export const levelIcon: Record<Level, string> = {good: '●', warn: '▲', bad: '✕', unknown: '○'}
export const levelLabel: Record<Level, Key> = {
  good: 'net.good',
  warn: 'net.warn',
  bad: 'net.bad',
  unknown: 'net.unknown',
}

// String-typed helpers for templates (Svelte 3 markup can't use TS casts).
export const icon = (l: string) => levelIcon[l as Level] ?? levelIcon.unknown
export const label = (l: string) => levelLabel[l as Level] ?? levelLabel.unknown

const rank: Record<Level, number> = {unknown: 0, good: 1, warn: 2, bad: 3}

export function worst(levels: string[]): Level {
  return (levels as Level[]).reduce<Level>((w, l) => (rank[l] > rank[w] ? l : w), 'unknown')
}

export function fmtMs(ms: number): string {
  // ICMP reports whole milliseconds, so 0 means under 1 ms.
  if (ms < 1) return '<1'
  return ms < 10 ? ms.toFixed(1) : Math.round(ms).toString()
}

// Diagnosis: the gateway separates a local (Wi-Fi/LAN) problem from an
// ISP/route problem.
export function diagnose(stats: NetStat[]): Key | null {
  const gw = stats.find((s) => s.target === 'gateway')
  const others = stats.filter((s) => s.target !== 'gateway')
  const othersBad = others.length > 0 && worst(others.map((s) => s.level)) !== 'good'
  if (gw && (gw.level === 'bad' || gw.level === 'warn')) return 'net.diagLocal'
  if (gw && gw.level === 'good' && othersBad) return 'net.diagRemote'
  return null
}
