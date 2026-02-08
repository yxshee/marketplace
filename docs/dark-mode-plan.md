# Dark Mode Plan

## Goal
Add a high-quality, token-based dark mode across buyer, vendor, and admin surfaces without changing backend behavior or introducing style duplication.

## Scope map

### Buyer surface
- `/`
- `/search`
- `/products/[productID]`
- `/cart`
- `/checkout`
- `/checkout/confirmation`

### Vendor surface
- `/vendor` dashboard and management sections

### Admin surface
- `/admin` operations and analytics sections

## Styling architecture
- Theme model: `light` / `dark` / `system`
- Tailwind strategy: `darkMode: "class"`
- Runtime theme provider: `next-themes`
- Token source of truth: `src/app/globals.css`
  - `:root` for light
  - `.dark` for dark
- Keep compatibility aliases during migration:
  - `--canvas -> --bg`
  - `--surface-soft -> --surface-2`
  - `--ink -> --text`
  - `--muted -> --text-muted`

## Token usage rules
- Backgrounds use `--bg`, `--bg-muted`, `--surface`, `--surface-2`.
- Text uses `--text` and `--text-muted`.
- Interactive accents use `--primary`, `--accent`.
- Borders use `--border`.
- Focus rings use `--ring`.
- Avoid hardcoded `#000`, `#fff`, `bg-white`, `ring-black/70`, and similar one-off overrides.

## Component migration targets
- Existing:
  - `src/components/ui/surface-card.tsx`
  - `src/components/catalog/product-card.tsx`
- New reusable primitives:
  - `src/components/providers/theme-provider.tsx`
  - `src/components/ui/theme-toggle.tsx`
  - `src/components/ui/button.tsx`
  - `src/components/ui/input.tsx`
  - `src/components/ui/select.tsx`
  - `src/components/ui/badge.tsx`
  - `src/components/ui/notice.tsx`

## Toggle placement
- Global toggle in root header (`src/app/layout.tsx`)
- Additional quick toggles:
  - Vendor page header actions
  - Admin page header actions

## Quality gates per PR
- `pnpm -r lint`
- `pnpm -r typecheck`
- `pnpm -r test`
- `pnpm -r build`
- Manual QA checklist:
  - no hydration warnings
  - no flash of wrong theme on first paint
  - toggle persists selection
  - keyboard focus visible
  - contrast acceptable on text and controls

## Risk controls
- Keep business logic untouched.
- Avoid wide refactors unrelated to dark mode.
- Use small PRs and scoped commits.
- Maintain compatibility aliases until all pages are tokenized.
