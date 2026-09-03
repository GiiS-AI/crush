---
name: giis-sales
description: Use when the user wants to find business leads, manage their GiiS sales pipeline (companies, contacts, activities), or send outreach/email campaigns through their own GiiS account. Calls GiiS's real REST API directly — requires a GiiS API key and an active paid plan for lead scraping and CRM access.
---

# GiiS Sales & CRM

Talks to the user's own GiiS account over its real REST API to find leads,
manage a CRM pipeline, and send outreach — all data lives in the user's
actual GiiS account, not a local file store.

## Authentication

```
Authorization: Bearer <GIIS_API_KEY>
```

Base URL: `https://chat.giis.ai/api`. The user creates an API key in the
GiiS web app under **Settings → Accounts & Access → API Keys**. Ask for it
(or an env var name that holds it) if it isn't already available — never
guess or fabricate a key.

## Plan requirements

- **Lead scraper** (`/scraper/*`) and **CRM** (`/crm/*`) require an active
  paid plan (Pro or Enterprise). A 402/403 response means the account is on
  the Free tier — tell the user plainly rather than retrying.
- **Email accounts / outreach** require Personal plan or higher.

## Finding leads

```
POST /scraper/keyword-search
{"keyword": "VP of Sales at Series B SaaS", "max_results": 10}
```

`max_results` is capped at 20 per request. The response is one of:

- Leads directly: `{"leads": [...], "search_engines_blocked": [...], "search_engines_failed": [...], "degraded": false}`
- A queued async job (when the user has a desktop app connected as a scrape node): `{"status": "queued", "job_id": "..."}` — poll `GET /scraper/keyword-search/{job_id}` until `status` is `completed` or `failed`.

Scrape a single URL directly:

```
POST /scraper/scrape
{"url": "https://example.com"}
```

List previously scraped leads:

```
GET /scraper/leads?status=unsynced&page=1&page_size=20
```

`status` filter is one of `unsynced`, `synced`, `converted`. Export as CSV
via `GET /scraper/leads/export/csv`.

## Managing the CRM

All CRM endpoints are scoped to the current user — there is no shared
team-wide CRM. Base path: `/crm`.

**Companies**

```
POST /crm/companies
{"name": "Acme Inc", "domain": "acme.com", "lifecycle_stage": "lead"}
```

`name` is the only required field. Optional: `domain`, `website`,
`industry`, `size_band`, `email`, `phone`, `lifecycle_stage` (free text,
defaults to `"lead"`), `source`, `notes`. List with
`GET /crm/companies?search=&lifecycle_stage=&source=&page=1&page_size=20`.
`GET/PATCH/DELETE /crm/companies/{company_id}` for a single record.

**Contacts**

```
POST /crm/contacts
{"company_id": "...", "first_name": "Sarah", "last_name": "Johnson", "email": "sarah@acme.com"}
```

At least one of `first_name`, `last_name`, `email`, `phone` is required (the
API rejects a contact with none of them). Same list/get/patch/delete
pattern as companies, under `/crm/contacts`.

**Turning a scraped lead into a contact**

```
POST /crm/contacts/from-lead/{lead_id}
```

`lead_id` comes from the scraper's `ScrapedLead.id` (the `id` field in a
`/scraper/*` response). No request body.

**Activities** (calls, emails, notes logged against a company or contact)

```
POST /crm/activities
{"contact_id": "...", "activity_type": "call", "subject": "Intro call", "outcome": "interested"}
```

`activity_type` is required and free text (e.g. `call`, `email`, `note`,
`meeting`). Exactly one of `company_id` or `contact_id` must be set — the
API rejects an activity with neither. Optional: `subject`, `content`,
`outcome`, `occurred_at`, `details` (arbitrary JSON object).

**Pipeline overview**

```
GET /crm/summary
```

**Approval tasks** (human-in-the-loop steps the CRM has queued, e.g. an
auto-drafted reply awaiting a decision):

```
GET /crm/approval-tasks
POST /crm/approval-tasks/{task_id}/decision
```

## Sending outreach

**Email accounts** (the user's own SMTP/IMAP sender) — `/email-accounts`:

```
POST /email-accounts
{
  "display_name": "Sales Team",
  "from_email": "outreach@company.com",
  "smtp_host": "smtp.gmail.com",
  "smtp_port": 587,
  "smtp_user": "outreach@company.com",
  "smtp_password": "app-password-here",
  "use_tls": true
}
```

A new account is unverified until `POST /email-accounts/{id}/verify` (sends
a real test email) succeeds. `POST /email-accounts/{id}/set-default` marks
it as the default sender. `PATCH /email-accounts/{id}` can flip
`requires_approval` (default `true`) — when `true`, the auto-responder
drafts replies to inbound leads but a human must approve each one before it
sends; setting it `false` lets the auto-responder send on its own.

**Campaigns** — one fixed subject/body template sent to a fixed recipient
list; there is no per-recipient AI personalization:

```
POST /outreach/campaigns
{
  "name": "Q1 SaaS outreach",
  "subject_template": "Quick question about {{company}}",
  "body_template": "Hi {{first_name}}, ...",
  "contact_ids": ["..."],
  "lead_ids": ["..."],
  "send_rate_per_hour": 100
}
```

Then `POST /outreach/campaigns/{campaign_id}/start` to begin sending, or
`.../pause` to pause. `GET /outreach/campaigns` lists all campaigns and
their `recipient_counts` (pending/sent/skipped/failed breakdown).
`GET /outreach/inbox`, `GET /outreach/sent`, and `GET /outreach/pending-replies`
+ `POST /outreach/pending-replies/{reply_id}/approve|reject` cover reply
handling when `requires_approval` is on.

## What this does not do

- There is no per-lead AI-personalized email generation — campaigns use one
  template for every recipient.
- Reply classification is exactly `interested | not_interested | question |
  out_of_office | bounce` — do not invent other categories.
- There is no multi-day automated follow-up sequence feature.
