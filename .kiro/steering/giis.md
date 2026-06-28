# GiiS-Code

Terminal AI agent for the GiiS platform.

Install: `curl -fsSL https://giis.ai/install-code | sh`

## Auth

```sh
export GIIS_SERVER_URL=https://chat.giis.ai
export GIIS_PAT=<personal access token from chat.giis.ai/user/tokens>
```

## Sales Workflow

When asked to find leads and send outreach, use these tools in order:

1. `giis_scrape` with `niche` and `city` parameters.
2. `giis_crm` with `action=push` for each lead.
3. `giis_email` with `lead_id` and `template=cold-intro`.

## GiiS Platform APIs

Base URL: `https://chat.giis.ai`

Endpoints:

- `POST /api/scraper/run` - scrape leads
- `POST /api/crm/leads` - push lead to CRM
- `GET /api/crm/leads` - list CRM leads
- `POST /api/outreach/send` - send cold email

Auth: `Authorization: Bearer <GIIS_PAT>`
