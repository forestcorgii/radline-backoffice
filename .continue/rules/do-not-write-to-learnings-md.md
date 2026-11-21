---
name: do-not-write-to-learnings-md
description: Prevent agents from writing to a standalone learnings.md file; route learnings to topic files in references/.
---

- Do not create or append to `references/learnings.md` or `references/00-index.md`.
- Instead, append discoveries to the bottom of the relevant topic markdown file in `references/` (e.g., `references/routing.md`, `references/ui-design-tokens.md`, `references/database-schema.md`).
- Format each entry under `### Context: <Topic Title>`, stating the **Problem** and **Enforced Solution**.