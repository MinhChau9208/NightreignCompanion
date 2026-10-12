import type {gamedata} from '../../wailsjs/go/models'
import type {Key} from './i18n'

export const damageTypes = ['standard', 'strike', 'slash', 'pierce', 'magic', 'fire', 'lightning', 'holy'] as const
export const statusTypes = ['poison', 'rot', 'bleed', 'frost', 'sleep', 'madness', 'blight'] as const

export type Version = 'normal' | 'everdark'

export const damageKey = (d: string) => `boss.dmg.${d}` as Key
export const statusKey = (s: string) => `boss.status.${s}` as Key
export const levelKey = (l: StatusLevel) => `boss.level.${l}` as Key

// Negation at or beyond these is called out as a weakness / resistance.
const weakAt = -10
const resistAt = 20

// weakAndResist sorts a form's damage types into what to bring and what to avoid.
export function weakAndResist(f: gamedata.BossForm) {
  const all = damageTypes.map((type) => ({type, value: f.negation[type] ?? 0}))
  return {
    weak: all.filter((d) => d.value <= weakAt).sort((a, b) => a.value - b.value),
    resist: all.filter((d) => d.value >= resistAt).sort((a, b) => b.value - a.value),
  }
}

// Status buildup thresholds come in three steps in the data (154/252/542).
export type StatusLevel = 'easy' | 'mid' | 'hard' | 'immune'

export function statusLevel(f: gamedata.BossForm, s: string): StatusLevel {
  const v = f.resist[s]
  if (v === undefined) return 'immune'
  if (v <= 154) return 'easy'
  if (v <= 252) return 'mid'
  return 'hard'
}

// easyStatuses lists the statuses with the lowest buildup threshold.
export const easyStatuses = (f: gamedata.BossForm) => statusTypes.filter((s) => statusLevel(f, s) === 'easy')

export const pct = (p: number) => (p >= 0.995 ? '100%' : p > 0 && p < 0.005 ? '<1%' : `${Math.round(p * 100)}%`)

export const signed = (v: number) => (v > 0 ? `+${v}` : `${v}`)

// Hint mirrors main.BossHint, published to the overlay as the "boss" event.
export type Hint = {nightlord: string; name: gamedata.Localized; probability: number; form: gamedata.BossForm | null}
