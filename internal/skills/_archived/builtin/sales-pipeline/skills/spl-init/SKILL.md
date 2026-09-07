---
name: spl-init
description: Scaffold the Sales Pipeline catalogs in a shared second-brain bundle.
---

# spl-init

Create the catalogs this plugin owns inside a shared knowledge root.

## Process

1. Confirm target (default `knowledge/`).
2. Run:

```bash
python3 ../../scripts/spl_common.py init-bundle \
  --bundle knowledge \
  --title "Sales Pipeline" \
  --catalogs "sales-leads,opportunities,deals,proposals,quotes,campaigns,touchpoints,objections,forecasts"
```

3. Point the user at `../../sample-knowledge/` for a fictional demo.

## Done when

- `knowledge/index.md` exists
- Each owned catalog has `index.md`
