# Redpanda UI Redesign — Design Spec
Date: 2026-06-11

## Goal
Replace the current GitHub-dark color scheme with a design that matches the Redpanda brand as seen on redpanda.com, cloud.redpanda.com, and docs.redpanda.com.

## Approved Design Direction
Dark navy sidebar + light main content area + Redpanda brand red accent. Confirmed via visual mockup.

## Color Palette

### CSS Variables (replacing current `:root`)
| Variable | Value | Usage |
|---|---|---|
| `--bg` | `#F9FAFB` | Page background |
| `--surface` | `#FFFFFF` | Card / panel backgrounds |
| `--surface-2` | `#F2F4F7` | Nested surfaces (e.g. log bg, filter inputs) |
| `--border` | `#EAECF0` | Card and input borders |
| `--text` | `#101828` | Primary text |
| `--muted` | `#667085` | Secondary / label text |
| `--accent` | `#E2401B` | Redpanda brand red — bars, active states, badges |
| `--accent-bg` | `#FEF2E9` | Light orange tint — tag bg, banner bg |
| `--accent-border` | `#F88639` | Medium orange — banner border, hover |
| `--green` | `#12B76A` | Hit badges, local mode indicator |
| `--sidebar-bg` | `#1D2939` | Sidebar background |
| `--sidebar-border` | `rgba(255,255,255,0.08)` | Sidebar dividers |
| `--sidebar-text` | `#98A2B3` | Sidebar nav links (inactive) |
| `--sidebar-text-active`| `#F88639` | Sidebar nav links (active) |
| `--sidebar-active-bg` | `rgba(226,64,27,0.18)` | Sidebar active nav item bg |

### Updated variables
- `--blue`: `#5D6B98` (indigo from Redpanda palette) — still referenced in JS for non-RP progress bars, download links, and inline code; keep the variable, update the value
- `--orange`: remove — not referenced in JS or HTML

## Component Changes

### Sidebar
- Background: `--sidebar-bg` (`#1D2939`)
- All borders: `--sidebar-border`
- Nav link text: `--sidebar-text`; active: `--sidebar-text-active` on `--sidebar-active-bg`
- Logo: a small `#E2401B` rounded square with bold italic "R", mirroring the Redpanda logomark
- Mode badge: green tint for Local, blue-gray tint for GH Pages — both use sidebar-appropriate colors

### Main content
- Body background: `--bg`
- Section title / headings: `--text` (near-black, no change to font weight/size)
- Muted labels: `--muted`

### Cards
- Background: `--surface` (white)
- Border: `1px solid var(--border)` with `border-radius: 10px`
- Slight box-shadow: `0 1px 3px rgba(0,0,0,0.06)` for lift

### Stat cards
- Value color: `--text`; accent stat (mention rate) uses `--accent`

### Progress bars
- Track: `--border`
- Fill: `--accent`

### Heatmap cells
- `.high`: `rgba(226,64,27,0.75)` bg, white text
- `.medium`: `rgba(226,64,27,0.35)` bg, `--text`
- `.low`: `rgba(226,64,27,0.15)` bg, `--muted`
- `.zero`: `--surface-2` bg, `--muted`

### Tags
- Default: `--accent-bg` bg, `#DA5D08` text (warm orange)
- `.redpanda`: same — it's already the accent
- Remove the `.blue` tag variant (replaced by default tag style)

### Badges
- `.badge-hit`: green tint — `rgba(18,183,106,0.12)` bg, `--green` text
- `.badge-miss`: `--surface-2` bg, `--muted` text
- `.badge-failed`: `--accent-bg` bg, `--accent` text
- `.badge-local`: `rgba(18,183,106,0.12)` bg, `--green` text
- `.badge-remote`: `#F2F4F7` bg, `--muted` text

### Buttons
- `.btn-primary`: `--accent` bg, white text
- `.btn-secondary`: `--surface-2` bg, `--text` text; hover: `--border` bg
- Both: `border-radius: 8px`

### New report banner
- Background: `--accent-bg`
- Border: `1px solid var(--accent-border)`
- Text: `#DA5D08`
- Button: `--accent` bg, white text

### Report selector (topbar)
- Move into a topbar bar: white bg, `--border` bottom border, padding `12px 24px`
- Select: white bg, `--border` border, `--text` color, `border-radius: 8px`

### Evidence cards
- Base: white + `--border`
- `.hit`: left border or border `rgba(18,183,106,0.4)` green tint
- `.failed`: opacity 0.6, no color change

### Filter input
- Background: `--surface`; border: `--border`; focus border: `--accent`

### Log output (terminal)
- Keep dark: background `#0C111D` (from Redpanda's darkest blue-gray), green text
- Border: `--border`

### Section title / headings
- Keep existing font sizes; `--text` color

## Sidebar Logo Treatment
Replace the 🔴 emoji with a styled `<span>` block: `background:#E2401B; border-radius:5px; color:#fff; font-weight:800; font-style:italic; padding: 2px 6px; font-size:13px` containing the letter **R**.

## Scope
- Changes are **CSS-only** (`dashboard.css`) plus the sidebar logo HTML in `index.html`
- No JS changes
- No layout/structure changes
- The dark log terminal intentionally stays dark

## Out of Scope
- Serif headings (would require a web font — not worth the load)
- Full gradient hero treatment (marketing site pattern, not appropriate for a data dashboard)
