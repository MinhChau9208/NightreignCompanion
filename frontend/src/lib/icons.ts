// Icons bundled from src/assets, keyed by the ids used in the data pack.
// A missing file (e.g. a Nightlord without a portrait) gives undefined.

const load = (files: Record<string, string>) =>
  Object.fromEntries(Object.entries(files).map(([path, url]) => [path.replace(/^.*\/|\.png$/g, ''), url]))

const damage = load(import.meta.glob('../assets/damage/*.png', {eager: true, import: 'default'}))
const status = load(import.meta.glob('../assets/status/*.png', {eager: true, import: 'default'}))
const nightlords = load(import.meta.glob('../assets/nightlords/*.png', {eager: true, import: 'default'}))

export const damageIcon = (d: string): string | undefined => damage[d]
export const statusIcon = (s: string): string | undefined => status[s]
export const portrait = (id: string): string | undefined => nightlords[id]
