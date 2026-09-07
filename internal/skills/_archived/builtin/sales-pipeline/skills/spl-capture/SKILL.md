---
name: spl-capture
description: Capture a Sales Pipeline noun into the shared second brain via the deterministic write helper.
---

# spl-capture

## Process

0. If more than one agent writes the shared brain, open an isolation session (`spl-session`) and export `SECOND_BRAIN_ROOT`.
   Claim identity `grok-bot/sales-pipeline` (or `deep-agents/sales-pipeline` on Deep Agents).
1. Identify the noun type from the allowed list (see `../../README.md`).
2. Collect title, status, author identity, and optional typed links.
3. Write with the helper — do not hand-author frontmatter unless the user insists:

```bash
python3 ../../scripts/spl_common.py write \
  --bundle knowledge \
  --type SalesLead \
  --folder sales-leads \
  --title "Example SalesLead" \
  --author "Grok Bot: Sales Pipeline" \
  --tags "spl"
```

4. Add typed links in a follow-up edit if needed (`rel` values from `../../docs/typed-edges.md`).
5. Validate.

Allowed types: SalesLead, Opportunity, Deal, Stage, NextAction, Objection, Competitor, Champion, EconomicBuyer, TechnicalBuyer, Proposal, Quote, Contract, NegotiationNote, WinLossReason, SalesCampaign, OutreachSequence, Touchpoint, ForecastEntry, PipelineSnapshot, ReferralSource.
