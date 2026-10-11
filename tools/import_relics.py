"""Regenerate data/relic_effects.json from the community spreadsheet (kept out
of git in data_source/).

    pip install openpyxl
    python tools/import_relics.py "data_source/Nightreign Useful Info (Version 1.03.2) (+ DLC).xlsx"

Reads the "Relic Effects", "Deep Relic Effects" and "Dormant Powers" sheets.
Effect names stay in English like the game; effect and note texts are
translated through tools/relic_vi.json, keyed by the English text. New or
changed texts are added there with an empty value and the script fails until
they are translated, so a patch never ships half-translated data. Rerun after
a game patch, then `go run ./cmd/nrc-cli validate`.
"""

import json
import re
import sys
from pathlib import Path

import openpyxl

ROOT = Path(__file__).resolve().parent.parent
VI = ROOT / "tools" / "relic_vi.json"

SHEETS = [
    # sheet, kind, column of the name, of the in-game text (None if absent)
    ("Relic Effects", "relic", 1, None),
    ("Deep Relic Effects", "deep", 1, None),
    ("Dormant Powers", "dormant", 1, 2),
]

CATEGORIES = {
    "stat": "stat",
    "start of game": "start",
    "exploration": "exploration",
    "offensive": "offensive",
    "defensive": "defensive",
    "regen": "regen",
    "character": "character",
    "curse": "curse",
}

STACKING = {"yes": "yes", "no": "no", "n/a": "na", "-": "na", "see notes": "varies"}

# Character effects whose name has no "[Character]" prefix.
CHARACTER_OVERRIDES = {
    "Defeating enemies near Totem Stela restores HP": "raider",
    "Improved Poise Near Totem Stela": "raider",
}

PREFIX = re.compile(r"^\[(\w+)\]")
# "[Consumable]" and "[Item]" are placeholders, not characters.
CHARACTERS = {"wylder", "guardian", "ironeye", "duchess", "raider", "revenant", "recluse", "executor",
              "scholar", "undertaker"}


def slug(s):
    return re.sub(r"[^a-z0-9]+", "_", s.lower().replace("'", "")).strip("_")


def clean(v):
    if v is None:
        return ""
    return re.sub(r"[ \t]+", " ", str(v)).strip()


def main(path):
    wb = openpyxl.load_workbook(path, data_only=True)
    vi = json.loads(VI.read_text(encoding="utf-8")) if VI.exists() else {}
    used, missing = set(), []

    def tr(en):
        used.add(en)
        if not vi.get(en):
            vi.setdefault(en, "")
            missing.append(en)
            return en
        return vi[en]

    out, ids = [], set()
    for sheet, kind, col_name, col_game in SHEETS:
        rows = list(wb[sheet].iter_rows(values_only=True))
        header = [clean(h).lower() for h in rows[0]]
        col_cat = header.index("category")
        col_effect = header.index("effect")
        col_stack = next(i for i, h in enumerate(header) if h.startswith("stackable"))
        col_notes = header.index("notes")
        for r in rows[1:]:
            name = clean(r[col_name])
            if not name:
                continue
            # Some dormant powers belong to two categories ("Defensive/Regen").
            cats = [CATEGORIES[c.strip().lower()] for c in re.split(r"[/\n]", str(r[col_cat])) if c.strip()]
            game = clean(r[col_game]) if col_game is not None else ""
            effect = clean(r[col_effect])
            if effect.lower() in ("self-explanatory", "self explanatory"):
                effect = ""  # the UI says "as named"
            elif not effect:
                effect = game
            m = PREFIX.match(name)
            character = m.group(1).lower() if m and m.group(1).lower() in CHARACTERS else ""
            character = character or CHARACTER_OVERRIDES.get(name, "")
            base = f"{kind}_{slug(name)}"
            rid, n = base, 2
            while rid in ids:
                rid, n = f"{base}_{n}", n + 1
            ids.add(rid)
            e = {
                "id": rid,
                "kind": kind,
                "categories": cats,
                "name": {"en": name, "vi": name},
            }
            if game:
                e["inGame"] = game
            if effect:
                e["effect"] = {"en": effect, "vi": tr(effect)}
            e["stacking"] = STACKING.get(clean(r[col_stack]).lower(), "unknown")
            notes = clean(r[col_notes])
            if notes:
                e["notes"] = {"en": notes, "vi": tr(notes)}
            if character:
                e["character"] = character
            e["verified"] = True
            out.append(e)

    # Keep the translation file in sheet order and drop texts no longer used.
    vi = {k: v for k, v in vi.items() if k in used}
    VI.write_text(json.dumps(vi, ensure_ascii=False, indent=2) + "\n", encoding="utf-8", newline="\n")
    if missing:
        raise SystemExit(f"{len(missing)} texts need a Vietnamese translation in {VI.relative_to(ROOT)}")

    dst = ROOT / "data" / "relic_effects.json"
    dst.write_text(json.dumps(out, ensure_ascii=False, indent=2) + "\n", encoding="utf-8", newline="\n")
    print("wrote", dst.relative_to(ROOT), f"({len(out)} effects)")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit(__doc__)
    main(sys.argv[1])
