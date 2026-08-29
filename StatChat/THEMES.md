# StatGate StatChat — Theme Specification

This document defines the theming system for the **StatChat** frontend. It
covers the design tokens, the two effective themes (Light / Dark), the optional
auto ("System") preference, and contrast guidance.

> Location: `StatChat/frontend/src/App.module.css` (`:root` tokens) with
> component-level overrides in each `*.module.css` and inline `theme`-prop
> conditionals in `App.tsx` + panels.

---

## 1. Theme preference model

StatChat supports three **user preferences**, persisted per user in
`user_settings.theme`:

| Preference | Meaning |
|---|---|
| `light` | Always use the Light theme |
| `dark` | Always use the Dark theme |
| `system` | Follow the OS `prefers-color-scheme` setting (resolved live) |

- `App.tsx` keeps a resolved effective theme (`light | dark`) that is passed to
  every component via the `theme` prop.
- When the preference is `system`, a `matchMedia('(prefers-color-scheme: dark)')`
  listener re-resolves the theme live on OS changes.
- Choice is persisted to the backend `user_settings.theme` (a plain `TEXT`
  column — no schema change required).

---

## 2. Design tokens (Light base)

Defined in `App.module.css :root`. These anchor the brand and surfaces:

| Token             | Value      | Usage |
|-------------------|------------|-------|
| `--primary-color` | `#165c92`  | Primary brand, buttons, actives |
| `--primary-light` | `#1a7ab5`  | Accents, links, hovers |
| `--primary-dark`  | `#0f3f5f`  | Dark header/sidebar surfaces |
| `--bg-primary`    | `#f9fafb`  | App background |
| `--bg-secondary`  | `#ffffff`  | Secondary surfaces |
| `--bg-card`       | `#ffffff`  | Cards / panels |
| `--border-color`  | `#e5e7eb`  | Borders and dividers |
| `--text-main`     | `#1a1a1a`  | Primary text |
| `--text-muted`    | `#6b7280`  | Secondary text |
| `--accent-cyan`   | `#165c92`  | Cyan-ish accent |
| `--accent-blue`   | `#1a70b0`  | Blue accent |
| `--accent-purple` | `#0f4170`  | Purple-ish accent |
| `--accent-gold`   | `#d99a19`  | Gold accent |
| `--accent-red`    | `#dc2626`  | Danger / destructive |
| `--accent-green`  | `#059669`  | Success / online |
| `--font-family`   | Inter + system fallbacks | Typeface |
| `--card-shadow`   | `0 2px 8px rgba(22,92,146,0.08)` | Card elevation |

---

## 3. Light theme

- **Background** `#f9fafb`; cards/panels **white** `#ffffff`.
- **Text** near-black `#1a1a1a`; muted `#6b7280`.
- **Borders** `#e5e7eb`.
- **Primary actions** `#165c92` (blue).
- **Active/hover highlights** `rgba(22,92,146,0.04–0.12)` blue tints.
- Header/canvas are white.

## 4. Dark theme

Applied via `headerDark` class + inline `theme === 'dark'` conditionals:

| Surface            | Color        |
|--------------------|--------------|
| Header gradient    | `#0a2b45 → #0f3f5f` |
| Chat headers / bars| `#0a2b45`    |
| Panels / inputs    | `#0f3f5f`    |
| Raised items       | `#0a324d`    |
| Text               | `#e8eef4` (pale blue-grey) |
| Borders            | `#6b7280`    |
| Modal overlay      | `rgba(8,25,38,0.56)` |

Brand blue `#165c92` remains the primary button colour in both modes.

---

## 5. Contrast guidance (WCAG 2.1 AA)

- Light: `#1a1a1a` on `#ffffff` → high contrast (✔).
- Dark: `#e8eef4` on `#0f3f5f`/`#0a2b45` → high contrast (✔).
- Muted text `#6b7280` should only be used for secondary metadata, not large
  bodies, to keep AA ratios.

---

## 6. Where the theme applies

- `theme` prop flows from `App.tsx` into all panels: `ChatSidebarList`,
  `ChatWorkspace`, `CollaborationPanel`, `CalendarPanel`, `SettingsPanel`,
  `KnowledgePanel`, `WellnessPanel`, `SidebarRail`, `IntegrationPanel`.
- Global tokens in `App.module.css` (`:root`), plus per-component modules
  (`*.module.css`) and inline `theme` conditionals for dynamic elements.