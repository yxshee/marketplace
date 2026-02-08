# Design System

## Principles
- Keep the interface clean and fast with a Gumroad-inspired editorial rhythm.
- Use one token source of truth for light and dark themes.
- Prefer composable primitives over page-local one-off styles.
- Preserve accessibility as a first-class requirement.

## Typography
- Display: `Sora`
- Body: `Manrope`
- Maintain strong hierarchy with `display-*` utilities in `tailwind.config.ts`.

## Shape and elevation
- Radius scale:
  - `--radius-xs: 10px`
  - `--radius-sm: 12px`
  - `--radius-md: 16px`
  - `--radius-lg: 22px`
- Shadows:
  - `--shadow-soft`
  - `--shadow-card`
  - `--shadow-float`
  - `--shadow-glow`

## Theme tokens

### Canonical semantic tokens
| Token | Light | Dark | Usage |
|---|---:|---:|---|
| `--bg` | `#FBF8FF` | `#120F1A` | App background |
| `--bg-muted` | `#F6F2FF` | `#181428` | Tinted sections |
| `--surface` | `#FFFFFF` | `#1D1829` | Cards and panels |
| `--surface-2` | `#F4EFFF` | `#262036` | Secondary containers |
| `--text` | `#1C1828` | `#F3F1FA` | Primary text |
| `--text-muted` | `#6D6881` | `#B6AFCB` | Secondary text |
| `--border` | `#DDD6EC` | `#3B334F` | Borders and separators |
| `--ring` | `#5967FF` | `#8B95FF` | Focus outlines |
| `--primary` | `#5967FF` | `#8B95FF` | Primary actions |
| `--primary-foreground` | `#FFFFFF` | `#0F1324` | Text on primary |
| `--accent` | `#EF4A9F` | `#FF6DBE` | Highlights/badges |
| `--accent-foreground` | `#FFFFFF` | `#2A1020` | Text on accent |
| `--success` | `#2EA66D` | `#46C98A` | Success state |
| `--warning` | `#DC9F1F` | `#F1B746` | Warning state |
| `--danger` | `#D94967` | `#F06A84` | Error/destructive |
| `--chart-1` | `#5967FF` | `#8B95FF` | Chart series 1 |
| `--chart-2` | `#EF4A9F` | `#FF6DBE` | Chart series 2 |
| `--chart-3` | `#2EA66D` | `#46C98A` | Chart series 3 |
| `--chart-4` | `#DC9F1F` | `#F1B746` | Chart series 4 |

### Compatibility aliases
To avoid broad regressions during migration:
- `--canvas -> --bg`
- `--surface-soft -> --surface-2`
- `--ink -> --text`
- `--muted -> --text-muted`

Existing brand and accent scales remain valid (`--brand-*`, `--accent-*`).

## Dark mode rules
- Theme strategy: Tailwind `darkMode: "class"` with `next-themes`.
- Default behavior: `system`.
- Manual selection: `light` / `dark` persists in local storage.
- All focus indicators must use tokenized ring color, not hardcoded black overlays.
- Avoid excessive `dark:` utility duplication; prefer semantic token classes.

## Layout and spacing
- Mobile-first with 4px rhythm.
- Max content width stays `6xl`.
- Preserve existing spacing behavior to avoid layout shift regressions.

## Component baseline
- Core surfaces:
  - Header navigation
  - Surface cards
  - Product cards
  - Form controls
  - Notice/alert blocks
- States:
  - default, hover, active, focus-visible, disabled, loading
