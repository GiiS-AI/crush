package cmd

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/client"
	"github.com/GiiS-AI/GiiS-Code/internal/clipboard"
	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/GiiS-AI/GiiS-Code/internal/oauth"
	"github.com/GiiS-AI/GiiS-Code/internal/oauth/claude"
	"github.com/GiiS-AI/GiiS-Code/internal/oauth/codex"
	"github.com/GiiS-AI/GiiS-Code/internal/oauth/copilot"
	"github.com/GiiS-AI/GiiS-Code/internal/oauth/hyper"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Aliases: []string{"auth"},
	Use:     "login [platform]",
	Short:   "Login GiiS-Code to a platform",
	Long: `Login GiiS-Code to a specified platform.
The platform should be provided as an argument.
Available platforms are: hyper, copilot, giis-cloud, claude, codex.`,
	Example: `
# Authenticate with Charm Hyper
c0d3r login

	# Authenticate with GitHub Copilot
	c0d3r login copilot

	# Authenticate with GiiS Cloud using your account credentials
	c0d3r login giis-cloud

	# Authenticate with Claude
	c0d3r login claude

	# Authenticate with Codex
	c0d3r login codex

	# Force re-authentication even if already logged in
	c0d3r login -f copilot
  `,
	ValidArgs: []cobra.Completion{
		"hyper",
		"copilot",
		"giis-cloud",
		"claude",
		"codex",
		"github",
		"github-copilot",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, ws, cleanup, err := connectToServer(cmd)
		if err != nil {
			return err
		}
		defer cleanup()

		progressEnabled := ws.Config.Options.Progress == nil || *ws.Config.Options.Progress
		if progressEnabled && supportsProgressBar() {
			_, _ = fmt.Fprintf(os.Stderr, ansi.SetIndeterminateProgressBar)
			defer func() { _, _ = fmt.Fprintf(os.Stderr, ansi.ResetProgressBar) }()
		}

		provider := "hyper"
		if len(args) > 0 {
			provider = args[0]
		}
		force, _ := cmd.Flags().GetBool("force")
		switch provider {
		case "hyper":
			return loginHyper(c, ws.ID, force)
		case "copilot", "github", "github-copilot":
			return loginCopilot(c, ws.ID, force)
		case "giis-cloud":
			return loginGiiSCloud(c, ws.ID, force)
		case "claude":
			return loginClaude(c, ws.ID, force)
		case "codex":
			return loginCodex(c, ws.ID, force)
		default:
			return fmt.Errorf("unknown platform: %s", provider)
		}
	},
}

func init() {
	loginCmd.Flags().BoolP("force", "f", false, "Force re-authentication even if already logged in")
}

func loginHyper(c *client.Client, wsID string, force bool) error {
	ctx := getLoginContext()

	if !force {
		cfg, err := c.GetConfig(ctx, wsID)
		if err == nil && cfg != nil {
			if pc, ok := cfg.Providers.Get("hyper"); ok && pc.OAuthToken != nil {
				fmt.Println("You are already logged in to Hyper.")
				fmt.Println("Use --force to re-authenticate.")
				return nil
			}
		}
	}

	resp, err := hyper.InitiateDeviceAuth(ctx)
	if err != nil {
		return err
	}

	clipboard.WriteText(resp.UserCode)
	fmt.Println("The following code should be on clipboard already:")

	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Render(resp.UserCode))
	fmt.Println()
	fmt.Println("Press enter to open this URL, and then paste it there:")
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Hyperlink(resp.VerificationURL, "id=hyper").Render(resp.VerificationURL))
	fmt.Println()
	waitEnter()
	if err := browser.OpenURL(resp.VerificationURL); err != nil {
		fmt.Println("Could not open the URL. You'll need to manually open the URL in your browser.")
	}

	fmt.Println("Exchanging authorization code...")
	refreshToken, err := hyper.PollForToken(ctx, resp.DeviceCode, resp.ExpiresIn)
	if err != nil {
		return err
	}

	fmt.Println("Exchanging refresh token for access token...")
	token, err := hyper.ExchangeToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	fmt.Println("Verifying access token...")
	introspect, err := hyper.IntrospectToken(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("token introspection failed: %w", err)
	}
	if !introspect.Active {
		return fmt.Errorf("access token is not active")
	}

	if err := cmp.Or(
		c.SetConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.api_key", token.AccessToken),
		c.SetConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with Hyper!")
	return nil
}

func loginCopilot(c *client.Client, wsID string, force bool) error {
	loginCtx := getLoginContext()

	if !force {
		cfg, err := c.GetConfig(loginCtx, wsID)
		if err == nil && cfg != nil {
			if pc, ok := cfg.Providers.Get("copilot"); ok && pc.OAuthToken != nil {
				fmt.Println("You are already logged in to GitHub Copilot.")
				fmt.Println("Use --force to re-authenticate.")
				return nil
			}
		}
	}

	diskToken, hasDiskToken := copilot.RefreshTokenFromDisk()
	var token *oauth.Token

	switch {
	case hasDiskToken:
		fmt.Println("Found existing GitHub Copilot token on disk. Using it to authenticate...")

		t, err := copilot.RefreshToken(loginCtx, diskToken)
		if err != nil {
			return fmt.Errorf("unable to refresh token from disk: %w", err)
		}
		token = t
	default:
		fmt.Println("Requesting device code from GitHub...")
		dc, err := copilot.RequestDeviceCode(loginCtx)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("Open the following URL and follow the instructions to authenticate with GitHub Copilot:")
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Hyperlink(dc.VerificationURI, "id=copilot").Render(dc.VerificationURI))
		fmt.Println()
		fmt.Println("Code:", lipgloss.NewStyle().Bold(true).Render(dc.UserCode))
		fmt.Println()
		fmt.Println("Waiting for authorization...")

		t, err := copilot.PollForToken(loginCtx, dc)
		if err == copilot.ErrNotAvailable {
			fmt.Println()
			fmt.Println("GitHub Copilot is unavailable for this account. To signup, go to the following page:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.SignupURL, "id=copilot-signup").Render(copilot.SignupURL))
			fmt.Println()
			fmt.Println("You may be able to request free access if eligible. For more information, see:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.FreeURL, "id=copilot-free").Render(copilot.FreeURL))
		}
		if err != nil {
			return err
		}
		token = t
	}

	if err := cmp.Or(
		c.SetConfigField(loginCtx, wsID, config.ScopeGlobal, "providers.copilot.api_key", token.AccessToken),
		c.SetConfigField(loginCtx, wsID, config.ScopeGlobal, "providers.copilot.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with GitHub Copilot!")
	return nil
}

func loginGiiSCloud(c *client.Client, wsID string, force bool) error {
	ctx := getLoginContext()

	if !force {
		cfg, err := c.GetConfig(ctx, wsID)
		if err == nil && cfg != nil {
			if pc, ok := cfg.Providers.Get("giis-cloud"); ok && pc.APIKey != "" {
				fmt.Println("You are already logged in to GiiS Cloud.")
				fmt.Println("Use --force to re-authenticate.")
				return nil
			}
		}
	}

	username, err := readLine("GiiS Cloud email: ")
	if err != nil {
		return err
	}
	password, err := readSecretLine("GiiS Cloud password: ")
	if err != nil {
		return err
	}
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	token, err := config.LoginCloudAndCreatePAT(ctx, username, password, "giis-code", nil)
	if err != nil {
		return err
	}

	if err := c.SetProviderAPIKey(ctx, wsID, config.ScopeGlobal, "giis-cloud", token.Token); err != nil {
		return err
	}
	if err := c.SetConfigField(ctx, wsID, config.ScopeGlobal, "providers.giis-cloud.provider_options.cloud_token_id", token.ID); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with GiiS Cloud!")
	return nil
}

func loginClaude(c *client.Client, wsID string, force bool) error {
	ctx := getLoginContext()

	if !force {
		cfg, err := c.GetConfig(ctx, wsID)
		if err == nil && cfg != nil {
			if pc, ok := cfg.Providers.Get("claude"); ok && pc.OAuthToken != nil {
				fmt.Println("You are already logged in to Claude.")
				fmt.Println("Use --force to re-authenticate.")
				return nil
			}
		}
	}

	fmt.Println("Prompting for Claude API key...")
	key, err := claude.PromptForAPIKey(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}

	fmt.Println("Validating Claude API key...")
	if err := claude.ValidateAPIKey(ctx, key); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	token := claude.TokenFromAPIKey(key)
	if err := c.SetConfigField(ctx, wsID, config.ScopeGlobal, "providers.claude.oauth", token); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with Claude!")
	return nil
}

func loginCodex(c *client.Client, wsID string, force bool) error {
	ctx := getLoginContext()

	if !force {
		cfg, err := c.GetConfig(ctx, wsID)
		if err == nil && cfg != nil {
			if pc, ok := cfg.Providers.Get("codex"); ok && pc.OAuthToken != nil {
				fmt.Println("You are already logged in to Codex.")
				fmt.Println("Use --force to re-authenticate.")
				return nil
			}
		}
	}

	fmt.Println("Prompting for Codex/OpenAI API key...")
	key, err := codex.PromptForAPIKey(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}

	fmt.Println("Validating Codex/OpenAI API key...")
	if err := codex.ValidateAPIKey(ctx, key); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	token := codex.TokenFromAPIKey(key)
	if err := c.SetConfigField(ctx, wsID, config.ScopeGlobal, "providers.codex.oauth", token); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with Codex!")
	return nil
}

func getLoginContext() context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	go func() {
		<-ctx.Done()
		cancel()
		os.Exit(1)
	}()
	return ctx
}

func waitEnter() {
	_, _ = fmt.Scanln()
}

func readSecretLine(prompt string) (string, error) {
	fmt.Print(prompt)
	if term.IsTerminal(os.Stdin.Fd()) {
		b, err := term.ReadPassword(os.Stdin.Fd())
		fmt.Println()
		return strings.TrimSpace(string(b)), err
	}

	var input string
	if _, err := fmt.Fscanln(os.Stdin, &input); err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	var input string
	if _, err := fmt.Fscanln(os.Stdin, &input); err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
