package giis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"charm.land/fantasy"
	giisskills "github.com/GiiS-AI/GiiS-Code/internal/skills"
)

const (
	SkillName = "giis-sales"

	serverURLEnv = "GIIS_SERVER_URL"
	patEnv       = "GIIS_PAT"

	scrapeToolName = "giis_scrape"
	emailToolName  = "giis_email"
	crmToolName    = "giis_crm"
)

const skillInstructions = `# GiiS Sales Workflow

Use these tools when the user asks you to find leads, contact them, or push them into CRM.

Workflow:
1. Use giis_scrape with niche and city.
2. Use giis_crm with action=push for each lead.
3. Use giis_email with lead_id and template=cold-intro.

Auth:
- GIIS_SERVER_URL: Base URL for the GiiS platform.
- GIIS_PAT: Personal access token used as a Bearer token.
`

type scrapeParams struct {
	Niche string `json:"niche" description:"The sales niche to scrape for."`
	City  string `json:"city" description:"The target city for the lead search."`
	Limit int    `json:"limit,omitempty" description:"Maximum number of leads to return. Defaults to 50."`
}

type emailParams struct {
	LeadID   string `json:"lead_id" description:"The lead ID to contact."`
	Template string `json:"template,omitempty" description:"The outreach template to use. Defaults to cold-intro."`
}

type crmParams struct {
	Action string `json:"action" description:"The CRM action to perform. Must be push or list."`
	LeadID string `json:"lead_id,omitempty" description:"The lead ID to push to CRM."`
}

func init() {
	giisskills.RegisterBuiltinProvider(func() ([]*giisskills.Skill, []*giisskills.SkillState) {
		if !Enabled() {
			return nil, nil
		}

		skill := &giisskills.Skill{
			Name:          SkillName,
			Description:   "Sales workflow tools for the GiiS platform.",
			Instructions:  skillInstructions,
			Path:          giisskills.BuiltinPrefix + SkillName,
			SkillFilePath: giisskills.BuiltinPrefix + SkillName + "/SKILL.md",
			Builtin:       true,
		}
		if err := skill.Validate(); err != nil {
			return nil, []*giisskills.SkillState{{Name: skill.Name, Path: skill.SkillFilePath, State: giisskills.StateError, Err: err}}
		}
		return []*giisskills.Skill{skill}, []*giisskills.SkillState{{Name: skill.Name, Path: skill.SkillFilePath, State: giisskills.StateNormal}}
	})
}

// Enabled reports whether the GiiS integration should be exposed.
func Enabled() bool {
	return strings.TrimSpace(os.Getenv(serverURLEnv)) != "" && strings.TrimSpace(os.Getenv(patEnv)) != ""
}

// Tools returns the GiiS sales tools when credentials are present.
func Tools() []fantasy.AgentTool {
	if !Enabled() {
		return nil
	}

	client := newHTTPClient()
	return []fantasy.AgentTool{
		newScrapeTool(client),
		newEmailTool(client),
		newCRMTool(client),
	}
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 20
	transport.MaxIdleConnsPerHost = 10
	transport.IdleConnTimeout = 90 * time.Second

	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}
}

func newScrapeTool(client *http.Client) fantasy.AgentTool {
	return fantasy.NewParallelAgentTool(
		scrapeToolName,
		"Scrape leads from the GiiS platform and return the JSON response.",
		func(ctx context.Context, params scrapeParams, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Niche == "" {
				return fantasy.NewTextErrorResponse("niche is required"), nil
			}
			if params.City == "" {
				return fantasy.NewTextErrorResponse("city is required"), nil
			}
			if params.Limit <= 0 {
				params.Limit = 50
			}

			body, err := json.Marshal(params)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			respBody, err := doJSONRequest(ctx, client, http.MethodPost, "/api/scraper/run", body)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}
			return fantasy.NewTextResponse(string(respBody)), nil
		},
	)
}

func newEmailTool(client *http.Client) fantasy.AgentTool {
	return fantasy.NewParallelAgentTool(
		emailToolName,
		"Send outreach to a lead through the GiiS platform.",
		func(ctx context.Context, params emailParams, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.LeadID == "" {
				return fantasy.NewTextErrorResponse("lead_id is required"), nil
			}
			if params.Template == "" {
				params.Template = "cold-intro"
			}

			body, err := json.Marshal(params)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to marshal request: %w", err)
			}

			respBody, err := doJSONRequest(ctx, client, http.MethodPost, "/api/outreach/send", body)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}
			return fantasy.NewTextResponse(string(respBody)), nil
		},
	)
}

func newCRMTool(client *http.Client) fantasy.AgentTool {
	return fantasy.NewParallelAgentTool(
		crmToolName,
		"Push leads into the GiiS CRM or list existing CRM leads.",
		func(ctx context.Context, params crmParams, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			switch params.Action {
			case "push":
				if params.LeadID == "" {
					return fantasy.NewTextErrorResponse("lead_id is required when action is push"), nil
				}
				body, err := json.Marshal(struct {
					LeadID string `json:"lead_id"`
				}{LeadID: params.LeadID})
				if err != nil {
					return fantasy.ToolResponse{}, fmt.Errorf("failed to marshal request: %w", err)
				}

				respBody, err := doJSONRequest(ctx, client, http.MethodPost, "/api/crm/leads", body)
				if err != nil {
					return fantasy.NewTextErrorResponse(err.Error()), nil
				}
				return fantasy.NewTextResponse(string(respBody)), nil

			case "list":
				respBody, err := doJSONRequest(ctx, client, http.MethodGet, "/api/crm/leads", nil)
				if err != nil {
					return fantasy.NewTextErrorResponse(err.Error()), nil
				}
				return fantasy.NewTextResponse(string(respBody)), nil

			default:
				return fantasy.NewTextErrorResponse("action must be push or list"), nil
			}
		},
	)
}

func doJSONRequest(ctx context.Context, client *http.Client, method, path string, body []byte) ([]byte, error) {
	if client == nil {
		client = newHTTPClient()
	}

	baseURL := strings.TrimSpace(os.Getenv(serverURLEnv))
	if baseURL == "" {
		return nil, fmt.Errorf("GIIS_SERVER_URL is required")
	}
	token := strings.TrimSpace(os.Getenv(patEnv))
	if token == "" {
		return nil, fmt.Errorf("GIIS_PAT is required")
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(baseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if len(respBody) == 0 {
		return []byte("null"), nil
	}
	if !json.Valid(respBody) {
		return nil, fmt.Errorf("response was not valid JSON: %s", strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}
