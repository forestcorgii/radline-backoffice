---
title: radline-shadcn-ui
description: Radline BackOffice shadcn/ui-inspired vanilla CSS token & component conventions (Go html/template + HTMX, no React/Tailwind CLI).
---

# Radline UI & shadcn Token Conventions

Radline BackOffice uses **Go `html/template` + HTMX + vanilla CSS** ([`static/index.css`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/static/index.css)). There is **no** `package.json`, `components.json`, React, or `shadcn` CLI.

- **Theme Tokens**: Always use HSL CSS variables (`hsl(var(--primary))`, `hsl(var(--border))`, `hsl(var(--muted))`) or mapped aliases (`var(--bg-color)`, `var(--surface-color)`, `var(--primary-color)`, `var(--text-main)`, `var(--text-muted)`, `var(--border-color)`). Never hardcode hex/RGB colors in templates.
- **Core Primitives**: Use existing classes in `static/index.css` — `.card`, `.btn` (`.btn-secondary`, `.btn-outline`, `.btn-ghost`, `.btn-danger`, `.btn-success`, `.btn-icon`), `.form-input`, `.search-filter-bar`, `.badge` (`.badge-active`, `.badge-inactive`, `.badge-secondary`), and `#shadcn-alert-dialog` (via `hx-confirm` or `window.showShadcnAlertDialog()`).
- **Full Reference**: Consult [`references/ui-design-tokens.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/ui-design-tokens.md) only when modifying CSS, layouts, or templates.
