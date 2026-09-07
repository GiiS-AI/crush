---
name: spl-pack
description: Build a bounded ContextPack from a Sales Pipeline root concept (default 2 hops, 20 nodes).
---

# spl-pack

## Process

```bash
python3 ../../scripts/spl_common.py pack \
  --bundle knowledge \
  --root "/sales-leads/example.md" \
  --hops 2 \
  --max-nodes 20
```

Use `--hops 1` for a tiny pack. Outbound edges only. Do not dump the whole tree.
