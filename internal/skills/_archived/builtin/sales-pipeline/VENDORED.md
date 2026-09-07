# Vendored Source

- Source repository: `https://github.com/SpillwaveSolutions/sales-pipeline`
- Source default branch: `main`
- Source commit: `da8727c5635c3759ac81fd06914984356cb818f9`
- Source commit date: `2026-08-22T00:19:35Z`
- Fetched on: `2026-09-04`
- License: `MIT` (`LICENSE` preserved verbatim)

# Imported Structure

This source repo is a multi-skill ContentPack, not a single `SKILL.md` folder.
To preserve the real upstream layout, the import keeps the upstream shared pack
structure under one new builtin directory:

- `skills/` with the six upstream skills: `spl-init`, `spl-capture`,
  `spl-pack`, `spl-validate`, `spl-session`, `spl-doctor`
- shared `scripts/`, `templates/`, `schemas/`, `commands/`,
  `sample-knowledge/`, `docs/typed-edges.md`
- attribution files: `README.md`, `CHANGELOG.md`, `LICENSE`

The only content edits were path adjustments in vendored `skills/*/SKILL.md`
files so c0d3r examples point at the bundled shared scripts/docs using relative
paths from each skill directory.

# Stripped Content

The following upstream content was intentionally not imported because c0d3r
discovers plain `SKILL.md` files and does not use host/plugin packaging,
marketplace metadata, or host hook manifests:

- `.claude-plugin/`
- `.codex-plugin/`
- `.cursor-plugin/`
- `.cursor/`
- `.grok-plugin/`
- `hooks/`
- `hosts/`
- `plugin.json`
- `marketplace.json`
- `package.json`
- `.work/`
- `CLAUDE.md`
- host-specific docs under `docs/` such as `CURSOR.md`, `GROK_BOT.md`,
  `HOSTS.md`, `LANG_CHAIN_DEEP_AGENTS.md`

Additional upstream general docs such as `docs/ONBOARDING.md`,
`docs/ISOLATION.md`, `docs/WORKLOG.md`, `docs/design.md`, and
`docs/user_guide/user-guide.md` were not imported because the vendored skills do
not reference them directly and the import was kept scoped to the skill pack's
runtime/support material.
