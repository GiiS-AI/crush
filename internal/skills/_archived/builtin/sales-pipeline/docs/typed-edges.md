# Typed edges — Sales Pipeline

Direction matters. Packs follow outbound edges by default.

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

Unknown `rel` values are treated as `info` by validation. Do not invent new names in this plugin.
