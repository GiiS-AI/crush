package codex

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
	apiURL      = "https://api.openai.com/v1/models"
	apiKeyPrefix = "sk-"
)

var ErrInvalidKey = errors.New("invalid Codex API key format")

// ReadStoredCredentials reads OpenAI tokens from ~/.codex/auth.json
func ReadStoredCredentials() (*oauth.Token, error) {
	authFile := filepath.Join(os.ExpandEnv("$HOME"), ".codex", "auth.json")

	data, err := os.ReadFile(authFile)
	if err != nil {
		return nil, fmt.Errorf("could not read Codex auth: %w", err)
	}

	var auth map[string]interface{}
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("could not parse Codex auth: %w", err)
	}

	tokens, ok := auth["tokens"].(map[string]interface{})
	if !ok {
		return nil, errors.New("tokens not found in auth")
	}

	accessToken, ok := tokens["access_token"].(string)
	if !ok || accessToken == "" {
		return nil, errors.New("access_token not found or empty in tokens")
	}

	token := &oauth.Token{
		AccessToken: accessToken,
		ExpiresAt:   time.Now().Add(87600 * time.Hour).Unix(), // 10 years
	}

	if refreshToken, ok := tokens["refresh_token"].(string); ok && refreshToken != "" {
		token.RefreshToken = refreshToken
	}

	return token, nil
}

// ValidateToken validates an OpenAI JWT access token
func ValidateToken(ctx context.Context, accessToken string) error {
	// OpenAI JWT tokens are inherently validated by them - we just need to verify it's not empty
	// and that we can make a basic API call with it
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("invalid or expired access token")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token validation failed: %s - %s", resp.Status, string(body))
	}

	return nil
}

// PromptForAPIKey prompts the user for a Codex/OpenAI API key.
func PromptForAPIKey(stdin io.Reader) (string, error) {
	reader := bufio.NewReader(stdin)
	fmt.Print("Enter your OpenAI API key (sk-...): ")

	key, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, apiKeyPrefix) {
		return "", fmt.Errorf("%w: must start with 'sk-'", ErrInvalidKey)
	}

	if len(key) < 20 {
		return "", fmt.Errorf("%w: key too short", ErrInvalidKey)
	}

	return key, nil
}

// ValidateAPIKey tests the Codex API key by calling the models endpoint.
func ValidateAPIKey(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API validation failed: %s - %s", resp.Status, string(body))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
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
