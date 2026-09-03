package cmd

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/signal"

	"charm.land/lipgloss/v2"
	"github.com/GiiS-AI/GiiS-Code/internal/client"
	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
)

var (
	logoutHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	logoutItemStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	logoutPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
)

var logoutCmd = &cobra.Command{
	Aliases: []string{"signout"},
	Use:     "logout [platform]",
	Short:   "Logout GiiS-Code from a platform",
	Long: `Logout GiiS-Code from a specified platform, removing stored credentials.
The platform should be provided as an argument.
If no argument is given, a list of logged-in platforms will be shown.
Available platforms are: hyper, copilot, giis-cloud, claude, codex.`,
	Example: `
# Sign out from c0d3r
c0d3r logout hyper

	# Sign out from GitHub Copilot
	c0d3r logout copilot

	# Sign out from GiiS Cloud
	c0d3r logout giis-cloud

	# Sign out from Claude
	c0d3r logout claude

	# Sign out from Codex
	c0d3r logout codex
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

		var provider string
		if len(args) == 0 {
			provider, err = pickLoggedInProvider(c, ws.ID)
			if err != nil {
				return err
			}
			if provider == "" {
				return nil
			}
		} else {
			provider = args[0]
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Print(logoutPromptStyle.Render(fmt.Sprintf("Are you sure you want to logout %s? (y/N) ", provider)))
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil || (response != "y" && response != "Y" && response != "yes" && response != "Yes" && response != "YES") {
				fmt.Println(logoutHeaderStyle.Render("Logout cancelled."))
				return nil
			}
		}

		switch provider {
		case "hyper":
			return logoutHyper(c, ws.ID)
		case "copilot", "github", "github-copilot":
			return logoutCopilot(c, ws.ID)
		case "giis-cloud":
			return logoutGiiSCloud(c, ws.ID)
		case "claude":
			return logoutClaude(c, ws.ID)
		case "codex":
			return logoutCodex(c, ws.ID)
		default:
			return fmt.Errorf("unknown platform: %s", provider)
		}
	},
}

func logoutHyper(c *client.Client, wsID string) error {
	ctx := getLogoutContext()

	if err := cmp.Or(
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.api_key"),
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.oauth"),
	); err != nil {
		return err
	}

	fmt.Println(logoutHeaderStyle.Render("Successfully logged out of Hyper."))
	return nil
}

func logoutCopilot(c *client.Client, wsID string) error {
	ctx := getLogoutContext()

	if err := cmp.Or(
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.copilot.api_key"),
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.copilot.oauth"),
	); err != nil {
		return err
	}

	fmt.Println(logoutHeaderStyle.Render("Successfully logged out of GitHub Copilot."))
	return nil
}

func logoutGiiSCloud(c *client.Client, wsID string) error {
	ctx := getLogoutContext()

	cfg, err := c.GetConfig(ctx, wsID)
	if err != nil {
		return err
	}
	pc, _ := cfg.Providers.Get("giis-cloud")
	apiKey := pc.APIKey

	var tokenID int
	if pc.ProviderOptions != nil {
		if raw, ok := pc.ProviderOptions["cloud_token_id"]; ok {
			switch v := raw.(type) {
			case float64:
				tokenID = int(v)
			case int:
				tokenID = v
			case int64:
				tokenID = int(v)
			}
		}
	}

	if tokenID == 0 && apiKey != "" {
		if id, ok, err := config.LookupCloudTokenID(ctx, apiKey); err == nil && ok {
			tokenID = id
		}
	}

	if tokenID != 0 && apiKey != "" {
		if err := config.RevokeCloudToken(ctx, apiKey, tokenID); err != nil {
			fmt.Println(logoutPromptStyle.Render(fmt.Sprintf("Could not revoke the server token (%v). Removing local credentials only.", err)))
		} else {
			fmt.Println(logoutHeaderStyle.Render("Successfully revoked the GiiS Cloud token."))
		}
	} else {
		fmt.Println(logoutPromptStyle.Render("Could not resolve the server token id. Removing local credentials only."))
	}

	if err := c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.giis-cloud.api_key"); err != nil {
		return err
	}
	if err := c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.giis-cloud.provider_options.cloud_token_id"); err != nil {
		return err
	}

	fmt.Println(logoutHeaderStyle.Render("Successfully logged out of GiiS Cloud."))
	return nil
}

func logoutClaude(c *client.Client, wsID string) error {
	ctx := getLogoutContext()

	if err := c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.claude.oauth"); err != nil {
		return err
	}

	fmt.Println(logoutHeaderStyle.Render("Successfully logged out of Claude."))
	return nil
}

func logoutCodex(c *client.Client, wsID string) error {
	ctx := getLogoutContext()

	if err := c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.codex.oauth"); err != nil {
		return err
	}

	fmt.Println(logoutHeaderStyle.Render("Successfully logged out of Codex."))
	return nil
}

func pickLoggedInProvider(c *client.Client, wsID string) (string, error) {
	ctx := getLogoutContext()

	cfg, err := c.GetConfig(ctx, wsID)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	type loggedInProvider struct {
		id   string
		name string
	}

	var loggedIn []loggedInProvider
	for p := range cfg.Providers.Seq() {
		if p.OAuthToken != nil || p.APIKey != "" {
			name := p.Name
			if name == "" {
				name = p.ID
			}
			loggedIn = append(loggedIn, loggedInProvider{id: p.ID, name: name})
		}
	}

	if len(loggedIn) == 0 {
		fmt.Println(logoutPromptStyle.Render("You are not logged in to any platform."))
		return "", nil
	}

	if len(loggedIn) == 1 {
		return loggedIn[0].id, nil
	}

	fmt.Println(logoutHeaderStyle.Render("Logged-in platforms:"))
	for i, p := range loggedIn {
		fmt.Println(logoutItemStyle.Render(fmt.Sprintf("  %d. %s", i+1, p.name)))
	}
	fmt.Print(logoutPromptStyle.Render(fmt.Sprintf("Select a platform to logout (1-%d): ", len(loggedIn))))

	var choice int
	_, err = fmt.Scanln(&choice)
	if err != nil || choice < 1 || choice > len(loggedIn) {
		fmt.Println(logoutHeaderStyle.Render("Logout cancelled."))
		return "", nil
	}

	return loggedIn[choice-1].id, nil
}

func init() {
	logoutCmd.Flags().BoolP("force", "f", false, "Skip logout confirmation prompt")
}

func getLogoutContext() context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	go func() {
		<-ctx.Done()
		cancel()
		os.Exit(1)
	}()
	return ctx
}
