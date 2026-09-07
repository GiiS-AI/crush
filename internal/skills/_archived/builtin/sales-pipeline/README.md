# Sales Pipeline

Sales pipeline ContentPack: leads, opportunities, stages, next actions, objections, proposals, and forecast entries.

MIT. Dual-host: **Claude Code**, **Grok Build**, and **Codex** (Agent Skill Standard). Writes OKF Markdown + YAML into a shared second-brain bundle so other agents and local jobs can read the same graph.

## Install

```bash
# Claude Code
/plugin marketplace add SpillwaveSolutions/sales-pipeline
/plugin install sales-pipeline@SpillwaveSolutions

# Skilz CLI
skilz install SpillwaveSolutions/sales-pipeline
```

Point the plugin at a shared knowledge root (default `knowledge/`). All sibling ContentPack plugins write into the same tree.

## Skills

| Skill | What it does |
|-------|----------------|
| `/spl-init` | Scaffold the catalogs this plugin owns |
| `/spl-capture` | Capture a noun into the shared second brain (deterministic write) |
| `/spl-pack` | Build a bounded ContextPack from a root concept |
| `/spl-validate` | Validate frontmatter, types, and links |
| `/spl-session` | Open or close an isolated write session (worktree + PR) |
| `/spl-doctor` | Health check of the bundle this plugin owns |

## Nouns this plugin may write

| Type | Meaning |
|------|---------|
| `SalesLead` | Named inbound or outbound lead |
| `Opportunity` | Qualified revenue chance |
| `Deal` | Actively negotiated opportunity |
| `Stage` | Pipeline stage definition |
| `NextAction` | Required next step with date |
| `Objection` | Buyer concern |
| `Competitor` | Competing alternative |
| `Champion` | Internal advocate |
| `EconomicBuyer` | Person who can spend |
| `TechnicalBuyer` | Person who evaluates fit |
| `Proposal` | Written offer |
| `Quote` | Priced offer |
| `Contract` | Legal agreement in play |
| `NegotiationNote` | Deal-desk note |
| `WinLossReason` | Why it closed |
| `SalesCampaign` | Outbound or inbound motion |
| `OutreachSequence` | Sequence of touches |
| `Touchpoint` | Single interaction |
| `ForecastEntry` | Period forecast line |
| `PipelineSnapshot` | Point-in-time pipeline view |
| `ReferralSource` | How the lead arrived |

## Relationships

| `rel` | Meaning |
|-------|---------|
| `belongs_to` | Lead belongs to client or campaign |
| `owned_by` | Pipeline owner identity |
| `originates_from` | Came from campaign or referral |
| `advances_to` | Moved to a stage |
| `blocked_by` | Stuck on objection or missing buyer |
| `competes_with` | Against a competitor |
| `proposed_as` | Has a proposal or quote |
| `related_to` | Soft association |
| `closed_as` | Won or lost with reason |

## Catalogs

- `sales-leads/`
- `opportunities/`
- `deals/`
- `proposals/`
- `quotes/`
- `campaigns/`
- `touchpoints/`
- `objections/`
- `forecasts/`

## Deterministic write boundary

The model proposes. Schema-enforced scripts commit:

```bash
python3 scripts/spl_common.py write \
  --bundle knowledge \
  --type SalesLead \
  --folder sales-leads \
  --title "Example" \
  --author "Grok Bot: Sales Pipeline"
```

Never invent `rel` values. Never write types owned by another plugin.



## Related plugins

### ContentPack suite

- [second-brain-core](https://github.com/SpillwaveSolutions/second-brain-core)
- [executive-coordination](https://github.com/SpillwaveSolutions/executive-coordination)
- [account-management](https://github.com/SpillwaveSolutions/account-management)
- [sales-pipeline](https://github.com/SpillwaveSolutions/sales-pipeline)
- [executive-job-search](https://github.com/SpillwaveSolutions/executive-job-search)
- [consulting-leads](https://github.com/SpillwaveSolutions/consulting-leads)
- [content-media](https://github.com/SpillwaveSolutions/content-media)
- [news-digest](https://github.com/SpillwaveSolutions/news-digest)
- [gtm-positioning](https://github.com/SpillwaveSolutions/gtm-positioning)
- [second-brain-marketplace](https://github.com/SpillwaveSolutions/second-brain-marketplace)
- [second-brain-starter](https://github.com/SpillwaveSolutions/second-brain-starter)

### Foundation

- [okf-plugin](https://github.com/SpillwaveSolutions/okf-plugin) — Open Knowledge Format graph engine
- [project-knowledge-capture](https://github.com/SpillwaveSolutions/project-knowledge-capture) — Project Knowledge Capture. The why second brain.
- [system-architecture-capture](https://github.com/SpillwaveSolutions/system-architecture-capture) — System Architecture Capture. The what-is-running second brain.
- [data-engineering-knowledge-capture](https://github.com/SpillwaveSolutions/data-engineering-knowledge-capture) — Data Engineering Knowledge Capture. The data-plane second brain.
- [wiki_ticket_sdd](https://github.com/SpillwaveSolutions/wiki_ticket_sdd) — WikiTicket SDD. Visible work log. Append-only ULID JSONL plus fold.
- [okf-agent-graph](https://github.com/SpillwaveSolutions/okf-agent-graph) — AGER. Orchestrator / Doer / Judge / Synthesizer.


## Onboarding

Grok Bot and other host agents should start at [docs/ONBOARDING.md](docs/ONBOARDING.md). That file is the history of the LLM-wiki effort, the destination state (Grok Bots and local agents sharing one git-native second brain), and the canonical public repo list.

## Multi-host

Works with Claude Code, Grok Build, Codex, Cursor, Agent Plugins 1.0 clients, Grok Bot, and LangChain Deep Agents.

| Host | How to load |
|------|-------------|
| Claude Code | marketplace + plugin install |
| Grok Build | zero-config Claude plugin |
| Codex | Agent Skills / `hooks/hooks.json` |
| Agent Plugins clients | root `plugin.json` + `skills/` |
| Grok Bot | [docs/GROK_BOT.md](docs/GROK_BOT.md) |
| Cursor | [docs/CURSOR.md](docs/CURSOR.md) — `.cursor-plugin` + Agent Plugins 1.0 |
| LangChain Deep Agents | [docs/LANG_CHAIN_DEEP_AGENTS.md](docs/LANG_CHAIN_DEEP_AGENTS.md) |

Write isolation (worktree + PR) lives in second-brain-core: [docs/ISOLATION.md](https://github.com/SpillwaveSolutions/second-brain-core/blob/main/docs/ISOLATION.md). Point `SECOND_BRAIN_ROOT` at the session bundle. Never hard-code a private remote.

Eight job-function plugins plus core. Knowledge root is always a local path or env the human already owns.

## License

MIT. Copyright 2026 Rick Hightower / contributors.
