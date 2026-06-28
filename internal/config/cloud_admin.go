package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"charm.land/catwalk/pkg/catwalk"
)

const cloudAdminProviderName = "openai-compatible"

type cloudAdminClient interface {
	UpsertProvider(context.Context, cloudLLMProviderUpsertRequest) error
	ListTokens(context.Context) ([]cloudPATResponse, error)
	DeleteToken(context.Context, int) error
}

type cloudSessionClient interface {
	Login(context.Context, string, string) error
	CreateToken(context.Context, string, *int) (cloudCreatedPATResponse, error)
}

type cloudPATResponse struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TokenDisplay string  `json:"token_display"`
	CreatedAt    string  `json:"created_at"`
	ExpiresAt    *string `json:"expires_at"`
	LastUsedAt   *string `json:"last_used_at"`
}

type cloudCreatedPATResponse struct {
	cloudPATResponse
	Token string `json:"token"`
}

type cloudCreateTokenRequest struct {
	Name           string `json:"name"`
	ExpirationDays *int   `json:"expiration_days,omitempty"`
}

type cloudLLMProviderUpsertRequest struct {
	Name                *string                            `json:"name,omitempty"`
	Provider            string                             `json:"provider"`
	APIKey              *string                            `json:"api_key,omitempty"`
	APIBase             *string                            `json:"api_base,omitempty"`
	IsPublic            bool                               `json:"is_public"`
	IsAutoMode          bool                               `json:"is_auto_mode"`
	Groups              []int                              `json:"groups,omitempty"`
	Personas            []int                              `json:"personas,omitempty"`
	DeploymentName      *string                            `json:"deployment_name,omitempty"`
	ID                  *int                               `json:"id,omitempty"`
	APIKeyChanged       bool                               `json:"api_key_changed,omitempty"`
	CustomConfigChanged bool                               `json:"custom_config_changed,omitempty"`
	ModelConfigurations []cloudModelConfigurationUpsertReq `json:"model_configurations,omitempty"`
}

type cloudModelConfigurationUpsertReq struct {
	Name               string  `json:"name"`
	IsVisible          bool    `json:"is_visible"`
	MaxInputTokens     *int64  `json:"max_input_tokens,omitempty"`
	SupportsImageInput *bool   `json:"supports_image_input,omitempty"`
	SupportsReasoning  *bool   `json:"supports_reasoning,omitempty"`
	DisplayName        *string `json:"display_name,omitempty"`
}

type realCloudAdminClient struct {
	baseURL string
	token   string
	http    *http.Client
}

type realCloudSessionClient struct {
	baseURL string
	http    *http.Client
}

func newRealCloudAdminClient() *realCloudAdminClient {
	return &realCloudAdminClient{
		baseURL: cloudAdminBaseURL(),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func newRealCloudSessionClient() *realCloudSessionClient {
	jar, _ := cookiejar.New(nil)
	return &realCloudSessionClient{
		baseURL: cloudAdminBaseURL(),
		http:    &http.Client{Timeout: 10 * time.Second, Jar: jar},
	}
}

func (c *realCloudAdminClient) UpsertProvider(ctx context.Context, req cloudLLMProviderUpsertRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal cloud provider request: %w", err)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/admin/llm/provider?is_creation=true", bytes.NewReader(body))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		r.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloud provider upsert failed: %s", resp.Status)
	}
	return nil
}

func (c *realCloudAdminClient) ListTokens(ctx context.Context) ([]cloudPATResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/user/pats", nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tokens failed: %s", resp.Status)
	}
	var tokens []cloudPATResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (c *realCloudAdminClient) DeleteToken(ctx context.Context, id int) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/user/pats/"+strconv.Itoa(id), nil)
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete token failed: %s", resp.Status)
	}
	return nil
}

func (c *realCloudSessionClient) Login(ctx context.Context, username, password string) error {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", username)
	form.Set("password", password)

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("cloud login failed: %s", resp.Status)
	}
	return nil
}

func (c *realCloudSessionClient) CreateToken(ctx context.Context, name string, expirationDays *int) (cloudCreatedPATResponse, error) {
	reqBody := cloudCreateTokenRequest{Name: name, ExpirationDays: expirationDays}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return cloudCreatedPATResponse{}, fmt.Errorf("marshal create token request: %w", err)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/user/pats", bytes.NewReader(body))
	if err != nil {
		return cloudCreatedPATResponse{}, err
	}
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(r)
	if err != nil {
		return cloudCreatedPATResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return cloudCreatedPATResponse{}, fmt.Errorf("create token failed: %s", resp.Status)
	}
	var out cloudCreatedPATResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return cloudCreatedPATResponse{}, err
	}
	return out, nil
}

func SyncBridgeProvider(ctx context.Context, token, bridgeURL string, models []catwalk.Model) error {
	if bridgeURL == "" {
		bridgeURL = bridgeBaseURL()
	}
	req := cloudLLMProviderUpsertRequest{
		Name:       strPtr("GiiS Bridge"),
		Provider:   cloudAdminProviderName,
		APIBase:    strPtr(bridgeURL),
		IsPublic:   true,
		IsAutoMode: false,
	}
	if len(models) > 0 {
		req.ModelConfigurations = make([]cloudModelConfigurationUpsertReq, 0, len(models))
		for _, m := range models {
			visible := true
			req.ModelConfigurations = append(req.ModelConfigurations, cloudModelConfigurationUpsertReq{
				Name:               m.ID,
				IsVisible:          visible,
				DisplayName:        strPtr(m.Name),
				SupportsImageInput: boolPtr(false),
				SupportsReasoning:  boolPtr(false),
			})
		}
	}
	client := &realCloudAdminClient{baseURL: cloudAdminBaseURL(), token: token, http: &http.Client{Timeout: 10 * time.Second}}
	return client.UpsertProvider(ctx, req)
}

// LookupCloudTokenID tries to resolve the numeric PAT ID for the given token.
// It is best-effort and falls back to false when no unique match is found.
func LookupCloudTokenID(ctx context.Context, token string) (int, bool, error) {
	client := &realCloudAdminClient{baseURL: cloudAdminBaseURL(), token: token, http: &http.Client{Timeout: 10 * time.Second}}
	tokens, err := client.ListTokens(ctx)
	if err != nil {
		return 0, false, err
	}
	if id, ok := cloudTokenIDFromDisplay(token, tokens); ok {
		return id, true, nil
	}
	return 0, false, nil
}

// LoginCloudAndCreatePAT authenticates with username/password and mints a PAT.
func LoginCloudAndCreatePAT(ctx context.Context, username, password, tokenName string, expirationDays *int) (cloudCreatedPATResponse, error) {
	client := newRealCloudSessionClient()
	if err := client.Login(ctx, username, password); err != nil {
		return cloudCreatedPATResponse{}, err
	}
	if tokenName == "" {
		tokenName = "giis-code"
	}
	return client.CreateToken(ctx, tokenName, expirationDays)
}

// RevokeCloudToken deletes a PAT by id using the given bearer token.
func RevokeCloudToken(ctx context.Context, token string, tokenID int) error {
	client := &realCloudAdminClient{baseURL: cloudAdminBaseURL(), token: token, http: &http.Client{Timeout: 10 * time.Second}}
	return client.DeleteToken(ctx, tokenID)
}

func cloudAdminBaseURL() string {
	return cloudRootURL()
}

func cloudRootURL() string {
	v := strings.TrimRight(cloudBaseURL(), "/")
	return strings.TrimSuffix(v, "/v1")
}

func cloudTokenIDFromDisplay(token string, tokens []cloudPATResponse) (int, bool) {
	cleanToken := normalizeToken(token)
	if cleanToken == "" {
		return 0, false
	}

	for _, t := range tokens {
		display := normalizeToken(t.TokenDisplay)
		if display == "" {
			continue
		}
		if display == cleanToken || strings.HasSuffix(cleanToken, display) || strings.HasSuffix(display, cleanToken) {
			return t.ID, true
		}
	}

	return 0, false
}

func normalizeToken(v string) string {
	v = strings.ToLower(v)
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, "_", "")
	return v
}

func strPtr(v string) *string { return &v }

func boolPtr(v bool) *bool { return &v }
