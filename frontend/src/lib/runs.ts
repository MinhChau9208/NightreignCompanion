import type {store} from '../../wailsjs/go/models'
import type {Key} from './i18n'

export type Run = store.Run

export const runResults = ['win', 'loss', 'abandoned'] as const
export const maxDepth = 5

export const resultKey = (r: string) => `runs.result.${r}` as Key

// What the predictor hands to the run form.
export type Prefill = {night1: string; night2: string; nightlord: string}

// parseClock reads "mm:ss", "h:mm:ss" or plain minutes; null if malformed,
// 0 for an empty field.
export function parseClock(text: string): number | null {
  const s = text.trim()
  if (!s) return 0
  if (/^\d+$/.test(s)) return Number(s) * 60
  const m = /^(?:(\d+):)?(\d{1,2}):(\d{2})$/.exec(s)
  if (!m || Number(m[3]) >= 60 || (m[1] !== undefined && Number(m[2]) >= 60)) return null
  return Number(m[1] ?? 0) * 3600 + Number(m[2]) * 60 + Number(m[3])
}

export type Group = {id: string; runs: number; wins: number; avgClear: number}

// groupBy tallies runs per non-empty key, most played first. avgClear only
// counts won runs with a recorded time (0 when there are none).
export function groupBy(runs: Run[], key: (r: Run) => string): Group[] {
  const m = new Map<string, {runs: number; wins: number; clear: number; timed: number}>()
  for (const r of runs) {
    const id = key(r)
    if (!id) continue
    const g = m.get(id) ?? {runs: 0, wins: 0, clear: 0, timed: 0}
    g.runs++
    if (r.result === 'win') {
      g.wins++
      if (r.clearSeconds > 0) {
        g.clear += r.clearSeconds
        g.timed++
      }
    }
    m.set(id, g)
  }
  return [...m]
    .map(([id, g]) => ({id, runs: g.runs, wins: g.wins, avgClear: g.timed ? Math.round(g.clear / g.timed) : 0}))
    .sort((a, b) => b.runs - a.runs || b.wins - a.wins)
}

export function summary(runs: Run[]) {
  const all = groupBy(runs, () => 'all')[0] ?? {id: 'all', runs: 0, wins: 0, avgClear: 0}
  return {...all, losses: runs.filter((r) => r.result === 'loss').length}
}
