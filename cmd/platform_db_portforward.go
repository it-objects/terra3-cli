package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/it-objects/terra3-cli/internal/api"
	"github.com/it-objects/terra3-cli/internal/config"
	"github.com/it-objects/terra3-cli/ssmclient"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	platformStageFlag     string
	platformPartitionFlag string
	platformLocalPortFlag int
	platformShowPassword  bool
)

var platformDbPortForwardCmd = &cobra.Command{
	Use:   "db-portforward",
	Short: "Open a localhost port-forward to a DB partition via the Terra3 control plane",
	Long: `Authenticates with Terra3 (no AWS profile required). The control plane starts an
SSM Session Manager port-forward to the environment bastion and returns opaque
session credentials. This CLI opens a local listener using those credentials.

If the Session Manager channel closes (e.g. idle timeout), a new session is
requested automatically while this command is running.

Break-glass: use 'terra3 db port-forward' with an AWS profile if you still have
Identity Center access to the workload account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformDbPortForward()
	},
}

func runPlatformDbPortForward() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.ApiUrl == "" {
		return fmt.Errorf("api_url not set in ~/.terra3/config.yaml")
	}

	client := api.NewClient(cfg.ApiUrl)

	stageID := platformStageFlag
	if stageID == "" {
		stages, err := client.ListDeveloperStages()
		if err != nil {
			return err
		}
		if len(stages) == 0 {
			return fmt.Errorf("no stages visible to you")
		}

		// Group stages by environment, preserving order of first appearance
		envGroups := make(map[string][]int)
		var envOrder []string
		for i, s := range stages {
			if _, exists := envGroups[s.EnvironmentID]; !exists {
				envOrder = append(envOrder, s.EnvironmentID)
			}
			envGroups[s.EnvironmentID] = append(envGroups[s.EnvironmentID], i)
		}

		// Resolve environment names (falls back to ID on error)
		envNames := make(map[string]string, len(envOrder))
		for _, envID := range envOrder {
			envNames[envID] = client.GetEnvironmentName(envID)
		}

		var displayLabels []string
		var displayToStage []int // -1 for section headers
		for _, envID := range envOrder {
			displayLabels = append(displayLabels, fmt.Sprintf("── %s ──", envNames[envID]))
			displayToStage = append(displayToStage, -1)
			for _, si := range envGroups[envID] {
				displayLabels = append(displayLabels, stages[si].Name)
				displayToStage = append(displayToStage, si)
			}
		}

		var selectedStageIdx int
		for {
			idx, err := promptSelect("Select stage", displayLabels)
			if err != nil {
				return err
			}
			if displayToStage[idx] == -1 {
				continue
			}
			selectedStageIdx = displayToStage[idx]
			break
		}
		stageID = stages[selectedStageIdx].ID
	}

	partitionName := platformPartitionFlag
	partitions, err := client.ListDbPartitions(stageID)
	if err != nil {
		return err
	}
	if len(partitions) == 0 {
		return fmt.Errorf("stage %s has no DB partitions", stageID)
	}
	if partitionName == "" {
		if len(partitions) == 1 {
			partitionName = partitions[0].Name
		} else {
			labels := make([]string, len(partitions))
			for i, p := range partitions {
				db := p.DatabaseName
				if db == "" {
					db = "primary"
				}
				labels[i] = fmt.Sprintf("%s  [%s]  db=%s user=%s", p.Name, db, p.SchemaName, p.Username)
			}
			idx, err := promptSelect("Select DB partition", labels)
			if err != nil {
				return err
			}
			partitionName = partitions[idx].Name
		}
	}

	var selected *api.DbPartition
	for i := range partitions {
		if partitions[i].Name == partitionName {
			selected = &partitions[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("partition %q not found on stage %s", partitionName, stageID)
	}

	localPort := platformLocalPortFlag
	if localPort == 0 {
		suggested := 15432
		if selected.DatabaseName == "secondary" {
			suggested = 13306
		}
		prompt := promptui.Prompt{
			Label:   fmt.Sprintf("Local port (default %d)", suggested),
			Default: strconv.Itoa(suggested),
			Validate: func(s string) error {
				n, err := strconv.Atoi(s)
				if err != nil || n < 1024 || n > 65535 {
					return fmt.Errorf("port must be 1024–65535")
				}
				return nil
			},
		}
		result, err := prompt.Run()
		if err != nil {
			return err
		}
		localPort, _ = strconv.Atoi(result)
	}

	if err := ensureLocalPortFree(localPort); err != nil {
		return err
	}

	if platformShowPassword {
		pw, err := client.GetDbPartitionPassword(stageID, partitionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not fetch password: %v\n", err)
		} else {
			fmt.Printf("Password: %s\n", pw)
		}
	}

	var currentSessionID string
	var interrupted bool

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		interrupted = true
		fmt.Fprintln(os.Stderr, "\nStopping…")
		if currentSessionID != "" {
			_ = client.EndDbTunnel(stageID, currentSessionID)
		}
		os.Exit(0)
	}()

	fmt.Printf("Opening tunnel to %s/%s on localhost:%d …\n", stageID, partitionName, localPort)
	fmt.Println("Press Ctrl+C to stop. Session will auto-reconnect on idle timeout.")

	for {
		tunnel, err := client.StartDbTunnel(stageID, partitionName, localPort)
		if err != nil {
			return err
		}
		currentSessionID = tunnel.SessionId

		fmt.Printf("\nConnected.\n")
		fmt.Printf("  Engine:   %s\n", tunnel.Engine)
		fmt.Printf("  Database: %s\n", tunnel.Database)
		fmt.Printf("  Username: %s\n", tunnel.Username)
		fmt.Printf("  Local:    localhost:%d → %s:%d\n", tunnel.LocalPort, tunnel.Host, tunnel.Port)
		fmt.Printf("  Session:  %s\n", tunnel.SessionId)
		fmt.Printf("\nConnect with your SQL client to localhost:%d\n\n", tunnel.LocalPort)

		sessionErr := ssmclient.RunSession(
			tunnel.Region,
			tunnel.SessionId,
			tunnel.StreamUrl,
			tunnel.TokenValue,
			tunnel.BastionInstanceId,
		)

		_ = client.EndDbTunnel(stageID, tunnel.SessionId)
		currentSessionID = ""

		if interrupted {
			return nil
		}

		if sessionErr != nil {
			fmt.Fprintf(os.Stderr, "session ended: %v\n", sessionErr)
		} else {
			fmt.Fprintln(os.Stderr, "session closed (idle timeout or remote end)")
		}

		fmt.Fprintln(os.Stderr, "Requesting a new session in 2s…")
		time.Sleep(2 * time.Second)
	}
}

func ensureLocalPortFree(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("local port %d is not available: %w", port, err)
	}
	return ln.Close()
}

func promptSelect(label string, items []string) (int, error) {
	prompt := promptui.Select{
		Label: label,
		Items: items,
		Size:  15,
	}
	idx, _, err := prompt.Run()
	return idx, err
}

func init() {
	platformCmd.AddCommand(platformDbPortForwardCmd)
	platformDbPortForwardCmd.Flags().StringVar(&platformStageFlag, "stage", "", "Stage ID (prompted if omitted)")
	platformDbPortForwardCmd.Flags().StringVar(&platformPartitionFlag, "partition", "", "DB partition name (prompted if omitted)")
	platformDbPortForwardCmd.Flags().IntVar(&platformLocalPortFlag, "local-port", 0, "Local port to listen on (prompted if omitted)")
	platformDbPortForwardCmd.Flags().BoolVar(&platformShowPassword, "show-password", false, "Print the partition password after connecting")
}
