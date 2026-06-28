package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/GiiS-AI/GiiS-Code/internal/bridge"
	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/spf13/cobra"
)

var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Run the local bridge",
	Long:  "Run the local bridge that keeps track of local model sources such as LM Studio, Ollama, Codex, and Claude Code. Use `c0d3r bridge sync-cloud` to publish the detected bridge models to GiiS Cloud.",
	Example: `# Run the local bridge
c0d3r bridge

# Publish local bridge models to GiiS Cloud
c0d3r bridge sync-cloud
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		interval, _ := cmd.Flags().GetDuration("refresh-interval")

		daemon := bridge.NewDaemon(addr, interval)
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

		fmt.Fprintf(cmd.OutOrStdout(), "GiiS bridge listening on http://%s\n", daemon.Addr)
		return daemon.Run(ctx)
	},
}

var bridgeDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect local models",
	RunE: func(cmd *cobra.Command, args []string) error {
		models, err := bridge.Detect(cmd.Context())
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			Models []bridge.LocalModel `json:"models"`
		}{Models: models})
	},
}

var bridgeSyncCloudCmd = &cobra.Command{
	Use:   "sync-cloud",
	Short: "Publish local bridge models to GiiS Cloud",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, ws, cleanup, err := connectToServer(cmd)
		if err != nil {
			return err
		}
		defer cleanup()

		cfg, err := c.GetConfig(cmd.Context(), ws.ID)
		if err != nil {
			return err
		}
		pc, ok := cfg.Providers.Get("giis-cloud")
		if !ok || pc.APIKey == "" {
			return fmt.Errorf("giis-cloud is not logged in; run `c0d3r login giis-cloud` first")
		}

		models, err := bridge.Detect(cmd.Context())
		if err != nil {
			return err
		}
		catwalkModels := make([]catwalk.Model, 0, len(models))
		for _, m := range models {
			catwalkModels = append(catwalkModels, catwalk.Model{ID: m.ID, Name: m.Name})
		}

		if err := config.SyncBridgeProvider(cmd.Context(), pc.APIKey, bridgeBaseURL(cmd), catwalkModels); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Synced %d local models to GiiS Cloud.\n", len(catwalkModels))
		return nil
	},
}

func init() {
	bridgeCmd.PersistentFlags().String("addr", "127.0.0.1:8787", "Address for the local bridge HTTP server")
	bridgeCmd.PersistentFlags().Duration("refresh-interval", 30*time.Second, "How often to refresh local model detection")
	bridgeCmd.AddCommand(bridgeDetectCmd)
	bridgeCmd.AddCommand(bridgeSyncCloudCmd)
	rootCmd.AddCommand(bridgeCmd)
}

func bridgeBaseURL(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("addr"); v != "" {
		return "http://" + v + "/v1"
	}
	return ""
}
