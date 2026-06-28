package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GiiS-AI/GiiS-Code/internal/oauth"
)

const (
	apiURL      = "https://api.openai.com/v1/models"
	apiKeyPrefix = "sk-"
)

var ErrInvalidKey = errors.New("invalid Codex API key format")

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
