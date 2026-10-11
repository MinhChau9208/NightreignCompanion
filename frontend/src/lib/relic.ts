import type {gamedata} from '../../wailsjs/go/models'
import type {Key} from './i18n'

export const relicKinds = ['relic', 'deep', 'dormant'] as const
export const relicCategories = ['stat', 'start', 'exploration', 'offensive', 'defensive', 'regen', 'character', 'curse'] as const

export const kindKey = (k: string) => `relic.kind.${k}` as Key
export const categoryKey = (c: string) => `relic.cat.${c}` as Key
export const stackKey = (s: string) => `relic.stack.${s}` as Key
export const kindHintKey = (k: string) => (k ? `relic.kindHint.${k}` : 'relic.intro') as Key
export const stackHintKey = (s: string) => `relic.stackHint.${s}` as Key

// fold lowercases and strips Vietnamese diacritics so "sat thuong" finds
// "sát thương".
export const fold = (s: string) =>
  s
    .toLowerCase()
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .replace(/đ/g, 'd')

// haystack is every text of an effect a search can match, in both languages.
export function haystack(e: gamedata.RelicEffect): string {
  return fold(
    [e.name.en, e.inGame ?? '', e.effect?.en ?? '', e.effect?.vi ?? '', e.notes?.en ?? '', e.notes?.vi ?? '', e.character ?? ''].join('\n'),
  )
}
