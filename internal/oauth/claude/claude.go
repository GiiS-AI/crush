package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GiiS-AI/GiiS-Code/internal/oauth"
)

const (
	apiURL      = "https://api.anthropic.com/v1/models"
	apiKeyPrefix = "sk-ant-"
)

var ErrInvalidKey = errors.New("invalid Claude API key format")

// ReadStoredCredentials reads Claude API key from ~/.claude/.credentials.json
func ReadStoredCredentials() (string, error) {
	credFile := filepath.Join(os.ExpandEnv("$HOME"), ".claude", ".credentials.json")

	data, err := os.ReadFile(credFile)
	if err != nil {
		return "", fmt.Errorf("could not read Claude credentials: %w", err)
	}

	var creds map[string]interface{}
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", fmt.Errorf("could not parse Claude credentials: %w", err)
	}

	// Navigate to claudeAiOauth.accessToken
	claudeAiOauth, ok := creds["claudeAiOauth"].(map[string]interface{})
	if !ok {
		return "", errors.New("claudeAiOauth not found in credentials")
	}

	accessToken, ok := claudeAiOauth["accessToken"].(string)
	if !ok || accessToken == "" {
		return "", errors.New("accessToken not found or empty in credentials")
	}

	return accessToken, nil
}

// PromptForAPIKey prompts the user for a Claude API key with masked input.
func PromptForAPIKey(stdin io.Reader) (string, error) {
	reader := bufio.NewReader(stdin)
	fmt.Print("Enter your Claude API key (sk-ant-...): ")

	key, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, apiKeyPrefix) {
		return "", fmt.Errorf("%w: must start with 'sk-ant-'", ErrInvalidKey)
	}

	if len(key) < 20 {
		return "", fmt.Errorf("%w: key too short", ErrInvalidKey)
	}

	return key, nil
}

// ValidateAPIKey tests a Claude credential by calling the models endpoint.
// ReadStoredCredentials returns an OAuth access token (Claude Pro/Max
// subscription, "sk-ant-oat01-..."), not a raw API key - those authenticate
// via "Authorization: Bearer", never "x-api-key" (that header is for actual
// API keys from console.anthropic.com and rejects OAuth tokens with 401,
// even though the token itself is valid). Confirmed against the real API.
func ValidateAPIKey(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API validation failed: %s", resp.Status)
	}

	return nil
}

// TokenFromAPIKey converts an API key to a Token for storage.
func TokenFromAPIKey(key string) *oauth.Token {
	return &oauth.Token{
		AccessToken: key,
		ExpiresAt:   time.Now().Add(87600 * time.Hour).Unix(), // 10 years
	}
}
