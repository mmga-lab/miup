package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/mmga-lab/miup/pkg/audit"
	"github.com/mmga-lab/miup/pkg/check"
	"github.com/mmga-lab/miup/pkg/cluster/executor"
	"github.com/mmga-lab/miup/pkg/cluster/manager"
	"github.com/mmga-lab/miup/pkg/cluster/spec"
	"github.com/mmga-lab/miup/pkg/component"
	"github.com/mmga-lab/miup/pkg/localdata"
	"github.com/mmga-lab/miup/pkg/logger"
	"github.com/mmga-lab/miup/pkg/playground"
	"github.com/mmga-lab/miup/pkg/version"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// auditLog logs an operation to the audit log
func auditLog(instance, command string, args []string, err error, duration time.Duration) {
	logger, logErr := audit.NewLogger()
	if logErr != nil {
		// Silently ignore audit log errors - don't fail the main operation
		return
	}

	status := audit.StatusSuccess
	errMsg := ""
	if err != nil {
		status = audit.StatusFailed
		errMsg = err.Error()
	}

	entry := &audit.Entry{
		Instance: instance,
		Command:  command,
		Args:     args,
		Status:   status,
		Duration: duration,
		Error:    errMsg,
	}

	// Ignore errors from logging - don't fail the main operation
	_ = logger.Log(entry)
}

var (
	verbose bool
	rootCmd = &cobra.Command{
		Use:   "miup",
		Short: "MiUp is a component manager for Milvus",
		Long: `MiUp is a component manager for Milvus vector database.

It provides commands for:
  - Installing and managing Milvus components
  - Deploying local development environments (playground)
  - Managing Milvus instances (local or Kubernetes)
  - Monitoring and diagnostics

Quick start:
  miup playground start    Start a local Milvus instance for development
  miup install birdwatcher Install Milvus ecosystem tool
  miup instance deploy     Deploy a Milvus instance

For more information, visit: https://github.com/mmga-lab/miup`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				logger.EnableDebug()
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newUninstallCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newPlaygroundCmd())
	rootCmd.AddCommand(newClusterCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newMirrorCmd())
	rootCmd.AddCommand(newBenchCmd())
}

func newVersionCmd() *cobra.Command {
	var short bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show miup version",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.GetVersionInfo()
			if short {
				fmt.Println(info.ShortString())
			} else {
				fmt.Println(info.String())
			}
		},
	}
	cmd.Flags().BoolVarP(&short, "short", "s", false, "Print short version")
	return cmd
}

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <component>[:<version>]",
		Short: "Install a Milvus ecosystem tool",
		Long: `Install a Milvus ecosystem tool from GitHub Releases.

Available components:
  birdwatcher     Milvus diagnostic and debugging tool (milvus-io/birdwatcher)
  milvus-backup   Milvus backup and restore utility (zilliztech/milvus-backup)

Version specification:
  - If no version is specified, the latest release will be installed
  - Use :<version> to install a specific version (e.g., birdwatcher:v1.1.0)

Examples:
  miup install birdwatcher              Install latest birdwatcher
  miup install birdwatcher:v1.1.0       Install specific version
  miup install milvus-backup            Install milvus-backup
  miup install birdwatcher milvus-backup   Install multiple components`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}
			if err := profile.InitProfile(); err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			mgr := component.NewManager(profile)

			for _, arg := range args {
				name, ver := parseComponentArg(arg)
				if err := mgr.Install(ctx, name, ver); err != nil {
					return fmt.Errorf("failed to install %s: %w", name, err)
				}
			}
			return nil
		},
	}
	return cmd
}

// parseComponentArg parses "component:version" format
func parseComponentArg(arg string) (name, version string) {
	parts := strings.SplitN(arg, ":", 2)
	name = parts[0]
	if len(parts) == 2 {
		version = parts[1]
	}
	return
}

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <component>[:<version>]",
		Short: "Uninstall a Milvus ecosystem tool",
		Long: `Uninstall a Milvus ecosystem tool.

If no version is specified, all versions of the component will be removed.

Examples:
  miup uninstall birdwatcher           Uninstall all versions of birdwatcher
  miup uninstall birdwatcher:v1.1.0    Uninstall specific version`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := component.NewManager(profile)

			for _, arg := range args {
				name, ver := parseComponentArg(arg)
				if err := mgr.Uninstall(ctx, name, ver); err != nil {
					return fmt.Errorf("failed to uninstall %s: %w", name, err)
				}
			}
			return nil
		},
	}
	return cmd
}

func newListCmd() *cobra.Command {
	var available bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed or available components",
		Long: `List installed or available Milvus ecosystem tools.

Examples:
  miup list              List all installed components
  miup list --available  List all available components`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if available {
				fmt.Println("Available components:")
				for name, def := range component.Registry {
					fmt.Printf("  %-15s %s (%s)\n", name, def.Description, def.Repo)
				}
				fmt.Println("\nInstall with: miup install <component>")
				return nil
			}

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := component.NewManager(profile)

			components, err := mgr.List(ctx)
			if err != nil {
				return err
			}

			if len(components) == 0 {
				fmt.Printf("No components installed (in %s)\n", profile.ComponentsDir())
				fmt.Println("\nAvailable components:")
				for name := range component.Registry {
					fmt.Printf("  miup install %s\n", name)
				}
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "COMPONENT\tVERSION\tINSTALLED\tPATH")
			for _, meta := range components {
				for ver, info := range meta.Versions {
					activeMarker := ""
					if ver == meta.Active {
						activeMarker = " (active)"
					}
					fmt.Fprintf(w, "%s\t%s%s\t%s\t%s\n",
						meta.Name,
						ver,
						activeMarker,
						info.InstalledAt.Format("2006-01-02"),
						info.BinaryPath,
					)
				}
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().BoolVar(&available, "available", false, "List available components")
	return cmd
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <component>[:<version>] [-- args...]",
		Short: "Run an installed component",
		Long: `Run an installed Milvus ecosystem tool.

If no version is specified, the active (most recently installed) version is used.
Use -- to separate miup flags from component arguments.

Examples:
  miup run birdwatcher                      Run birdwatcher (active version)
  miup run birdwatcher:v1.1.0               Run specific version
  miup run birdwatcher -- connect etcd      Pass arguments to birdwatcher`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			// Parse component and version
			name, ver := parseComponentArg(args[0])

			// Args after -- are passed to the component
			componentArgs := args[1:]
			if cmd.ArgsLenAtDash() > 0 {
				componentArgs = args[cmd.ArgsLenAtDash():]
			}

			mgr := component.NewManager(profile)
			return mgr.Run(ctx, name, ver, componentArgs)
		},
	}
	return cmd
}

func newPlaygroundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playground",
		Short: "Manage local Milvus playground",
		Long: `Playground provides a quick way to start a local Milvus instance for development and testing.

Examples:
  miup playground start              Start default standalone Milvus
  miup playground start --mode cluster   Start cluster mode
  miup playground start --with-monitor   Start with Prometheus and Grafana
  miup playground stop               Stop the playground
  miup playground status             Show playground status
  miup playground list               List all playground instances`,
	}

	cmd.AddCommand(newPlaygroundStartCmd())
	cmd.AddCommand(newPlaygroundStopCmd())
	cmd.AddCommand(newPlaygroundStatusCmd())
	cmd.AddCommand(newPlaygroundListCmd())
	cmd.AddCommand(newPlaygroundLogsCmd())
	cmd.AddCommand(newPlaygroundCleanCmd())

	return cmd
}

func newPlaygroundStartCmd() *cobra.Command {
	var (
		tag         string
		withMonitor bool
		milvusVer   string
		milvusPort  int
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a local Milvus playground (standalone mode)",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}
			if err := profile.InitProfile(); err != nil {
				return err
			}

			// Set default tag if not provided
			if tag == "" {
				tag = "default"
			}

			// Create configuration
			cfg := playground.DefaultConfig()
			cfg.Tag = tag
			cfg.WithMonitor = withMonitor
			if milvusVer != "latest" && milvusVer != "" {
				cfg.MilvusVersion = milvusVer
			}
			if milvusPort != 0 {
				cfg.MilvusPort = milvusPort
			}

			// Create context with signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			// Start playground
			manager := playground.NewManager(profile)
			if err := manager.Start(ctx, cfg); err != nil {
				return err
			}

			// Print connection info
			fmt.Println()
			fmt.Println("Connect to Milvus:")
			fmt.Printf("  %s\n", color.CyanString("Endpoint: localhost:%d", cfg.MilvusPort))
			fmt.Printf("  %s\n", color.CyanString("SDK:      from pymilvus import MilvusClient"))
			fmt.Printf("  %s\n", color.CyanString("          client = MilvusClient('http://localhost:%d')", cfg.MilvusPort))
			if withMonitor {
				fmt.Println()
				fmt.Println("Monitoring:")
				fmt.Printf("  %s\n", color.CyanString("Prometheus: http://localhost:%d", cfg.PrometheusPort))
				fmt.Printf("  %s\n", color.CyanString("Grafana:    http://localhost:%d (admin/admin)", cfg.GrafanaPort))
			}
			fmt.Println()
			fmt.Printf("MinIO Console: %s\n", color.CyanString("http://localhost:%d (minioadmin/minioadmin)", cfg.MinioConsole))

			return nil
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "default", "Tag name for the playground instance")
	cmd.Flags().BoolVar(&withMonitor, "with-monitor", false, "Start with Prometheus and Grafana")
	cmd.Flags().StringVar(&milvusVer, "milvus.version", "latest", "Milvus version to use")
	cmd.Flags().IntVar(&milvusPort, "port", 19530, "Milvus port")

	return cmd
}

func newPlaygroundStopCmd() *cobra.Command {
	var (
		tag           string
		removeVolumes bool
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the Milvus playground",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			if tag == "" {
				tag = "default"
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			manager := playground.NewManager(profile)
			return manager.Stop(ctx, tag, removeVolumes)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "default", "Tag name of the playground instance to stop")
	cmd.Flags().BoolVar(&removeVolumes, "volumes", false, "Remove data volumes")

	return cmd
}

func newPlaygroundStatusCmd() *cobra.Command {
	var tag string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show playground status",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			if tag == "" {
				tag = "default"
			}

			ctx := context.Background()
			manager := playground.NewManager(profile)

			status, err := manager.Status(ctx, tag)
			if err != nil {
				return err
			}

			fmt.Printf("Playground: %s\n", color.CyanString(tag))
			fmt.Printf("Status:     %s\n", formatStatus(status.Status))
			fmt.Printf("Mode:       %s\n", status.Meta.Mode)
			fmt.Printf("Version:    %s\n", status.Meta.MilvusVersion)
			fmt.Printf("Port:       %d\n", status.Meta.MilvusPort)
			fmt.Printf("Created:    %s\n", status.Meta.CreatedAt.Format("2006-01-02 15:04:05"))

			if status.Status == playground.StatusRunning && status.ContainerStatus != "" {
				fmt.Println()
				fmt.Println("Containers:")
				fmt.Println(status.ContainerStatus)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "default", "Tag name of the playground instance")

	return cmd
}

func newPlaygroundListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all playground instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			manager := playground.NewManager(profile)

			instances, err := manager.List(ctx)
			if err != nil {
				return err
			}

			if len(instances) == 0 {
				fmt.Println("No playground instances found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TAG\tSTATUS\tMODE\tVERSION\tPORT\tCREATED")

			for _, inst := range instances {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
					inst.Meta.Tag,
					inst.Status,
					inst.Meta.Mode,
					inst.Meta.MilvusVersion,
					inst.Meta.MilvusPort,
					inst.Meta.CreatedAt.Format("2006-01-02 15:04"),
				)
			}

			w.Flush()
			return nil
		},
	}
	return cmd
}

func newPlaygroundLogsCmd() *cobra.Command {
	var (
		tag     string
		service string
		tail    int
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show playground logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			if tag == "" {
				tag = "default"
			}

			ctx := context.Background()
			manager := playground.NewManager(profile)

			logs, err := manager.Logs(ctx, tag, service, tail)
			if err != nil {
				return err
			}

			fmt.Print(logs)
			return nil
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "default", "Tag name of the playground instance")
	cmd.Flags().StringVarP(&service, "service", "s", "", "Service name (e.g., standalone, etcd, minio)")
	cmd.Flags().IntVarP(&tail, "tail", "n", 100, "Number of lines to show")

	return cmd
}

func newPlaygroundCleanCmd() *cobra.Command {
	var tag string

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up playground instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			if tag == "" {
				tag = "default"
			}

			ctx := context.Background()
			manager := playground.NewManager(profile)

			return manager.Clean(ctx, tag)
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "default", "Tag name of the playground instance to clean")

	return cmd
}

func formatStatus(status playground.Status) string {
	switch status {
	case playground.StatusRunning:
		return color.GreenString("running")
	case playground.StatusStopped:
		return color.YellowString("stopped")
	default:
		return color.RedString("unknown")
	}
}

func newClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Manage Milvus instances",
		Long: `Instance provides commands for deploying and managing Milvus instances on Kubernetes.

Kubernetes deployment uses Milvus Operator (supports standalone and distributed modes).

Examples:
  miup instance deploy prod topology.yaml              Deploy to Kubernetes
  miup instance list                                   List all instances
  miup instance display prod                           Show instance details
  miup instance start prod                             Start an instance
  miup instance stop prod                              Stop an instance
  miup instance scale prod --component querynode --replicas 3   Scale a component
  miup instance replicas prod                          Show current replicas
  miup instance upgrade prod v2.5.5                    Upgrade to a new version
  miup instance config show prod                       Show configuration
  miup instance config set prod key=value              Set configuration
  miup instance diagnose prod                          Health diagnostics
  miup instance destroy prod                           Destroy an instance
  miup instance check                                  Pre-deployment environment check`,
	}

	cmd.AddCommand(newInstanceCheckCmd())
	cmd.AddCommand(newInstanceAuditCmd())
	cmd.AddCommand(newInstanceDeployCmd())
	cmd.AddCommand(newInstanceListCmd())
	cmd.AddCommand(newInstanceDisplayCmd())
	cmd.AddCommand(newInstanceStartCmd())
	cmd.AddCommand(newInstanceStopCmd())
	cmd.AddCommand(newInstanceScaleCmd())
	cmd.AddCommand(newInstanceReplicasCmd())
	cmd.AddCommand(newInstanceUpgradeCmd())
	cmd.AddCommand(newInstanceConfigCmd())
	cmd.AddCommand(newInstanceDiagnoseCmd())
	cmd.AddCommand(newInstanceDestroyCmd())
	cmd.AddCommand(newInstanceLogsCmd())
	cmd.AddCommand(newInstanceTemplateCmd())

	return cmd
}

func newInstanceDeployCmd() *cobra.Command {
	var (
		skipConfirm   bool
		milvusVersion string
		kubeconfig    string
		kubecontext   string
		namespace     string
		withMonitor   bool
	)

	cmd := &cobra.Command{
		Use:   "deploy <instance-name> <topology.yaml>",
		Short: "Deploy a Milvus instance to Kubernetes",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			topoFile := args[1]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}
			if err := profile.InitProfile(); err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			mgr := manager.NewManager(profile)
			opts := manager.DeployOptions{
				MilvusVersion: milvusVersion,
				SkipConfirm:   skipConfirm,
				Kubeconfig:    kubeconfig,
				KubeContext:   kubecontext,
				Namespace:     namespace,
				WithMonitor:   withMonitor,
			}

			start := time.Now()
			deployErr := mgr.Deploy(ctx, instanceName, topoFile, opts)
			auditLog(instanceName, "deploy", []string{topoFile}, deployErr, time.Since(start))
			if deployErr != nil {
				return deployErr
			}

			// Print connection info
			info, _ := mgr.Display(ctx, instanceName)
			if info != nil && info.Meta != nil {
				ns := namespace
				if ns == "" {
					ns = info.Meta.Namespace
				}
				fmt.Println()
				fmt.Println("Connect to Milvus:")
				fmt.Printf("  %s\n", color.CyanString("Namespace: %s", ns))
				fmt.Printf("  %s\n", color.CyanString("Use: kubectl port-forward svc/%s-milvus -n %s 19530:19530", instanceName, ns))
				fmt.Printf("  %s\n", color.CyanString("SDK:      from pymilvus import MilvusClient"))
				fmt.Printf("  %s\n", color.CyanString("          client = MilvusClient('http://localhost:19530')"))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation")
	cmd.Flags().StringVar(&milvusVersion, "milvus.version", "v2.5.4", "Milvus version to use")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (defaults to ~/.kube/config)")
	cmd.Flags().StringVar(&kubecontext, "context", "", "Kubernetes context to use")
	cmd.Flags().StringVar(&namespace, "namespace", "milvus", "Kubernetes namespace for deployment")
	cmd.Flags().BoolVar(&withMonitor, "with-monitor", false, "Enable monitoring (creates PodMonitor for Prometheus Operator)")

	return cmd
}

func newInstanceListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := manager.NewManager(profile)

			instances, err := mgr.List(ctx)
			if err != nil {
				return err
			}

			if len(instances) == 0 {
				fmt.Println("No instances found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATUS\tMODE\tBACKEND\tVERSION\tPORT\tCREATED")

			for _, c := range instances {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
					c.Name,
					c.Status,
					c.Mode,
					c.Backend,
					c.MilvusVersion,
					c.MilvusPort,
					c.CreatedAt.Format("2006-01-02 15:04"),
				)
			}

			w.Flush()
			return nil
		},
	}
	return cmd
}

func newInstanceDisplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display <instance-name>",
		Short: "Display instance details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := manager.NewManager(profile)

			info, err := mgr.Display(ctx, instanceName)
			if err != nil {
				return err
			}

			meta := info.Meta
			fmt.Printf("Cluster:  %s\n", color.CyanString(meta.Name))
			fmt.Printf("Status:   %s\n", formatClusterStatus(meta.Status))
			fmt.Printf("Mode:     %s\n", meta.Mode)
			fmt.Printf("Backend:  %s\n", meta.Backend)
			fmt.Printf("Version:  %s\n", meta.MilvusVersion)
			fmt.Printf("Port:     %d\n", meta.MilvusPort)
			fmt.Printf("Created:  %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))

			if info.ContainerStatus != "" {
				fmt.Println()
				fmt.Println("Containers:")
				fmt.Println(info.ContainerStatus)
			}

			return nil
		},
	}
	return cmd
}

func newInstanceStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <instance-name>",
		Short: "Start an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mgr := manager.NewManager(profile)
			start := time.Now()
			startErr := mgr.Start(ctx, instanceName)
			auditLog(instanceName, "start", nil, startErr, time.Since(start))
			return startErr
		},
	}
	return cmd
}

func newInstanceStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <instance-name>",
		Short: "Stop an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mgr := manager.NewManager(profile)
			start := time.Now()
			stopErr := mgr.Stop(ctx, instanceName)
			auditLog(instanceName, "stop", nil, stopErr, time.Since(start))
			return stopErr
		},
	}
	return cmd
}

func newInstanceScaleCmd() *cobra.Command {
	var (
		component     string
		replicas      int
		cpuRequest    string
		cpuLimit      string
		memoryRequest string
		memoryLimit   string
	)

	cmd := &cobra.Command{
		Use:   "scale <instance-name>",
		Short: "Scale a component in the instance",
		Long: `Scale a Milvus component by changing replicas and/or resources.

This command only works with Kubernetes deployments (distributed mode).
Local deployments (standalone mode) do not support scaling.

You can perform:
  - Horizontal scaling: change the number of replicas
  - Vertical scaling: change CPU/memory resources

Available components for distributed mode:
  proxy       Milvus proxy (API gateway)
  querynode   Query node (handles search requests)
  datanode    Data node (handles data writes)
  indexnode   Index node (builds indexes)
  rootcoord   Root coordinator
  querycoord  Query coordinator
  datacoord   Data coordinator
  indexcoord  Index coordinator

Examples:
  # Scale replicas (horizontal scaling)
  miup instance scale prod --component querynode --replicas 5
  miup instance scale prod -c datanode -r 3

  # Update resources (vertical scaling)
  miup instance scale prod -c querynode --cpu-request 2 --memory-request 8Gi
  miup instance scale prod -c querynode --cpu-limit 4 --memory-limit 16Gi

  # Combined scaling (both replicas and resources)
  miup instance scale prod -c querynode -r 5 --cpu-request 4 --memory-request 16Gi`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			if component == "" {
				return fmt.Errorf("--component is required")
			}

			// Build scale options
			opts := executor.ScaleOptions{
				Replicas:      replicas,
				CPURequest:    cpuRequest,
				CPULimit:      cpuLimit,
				MemoryRequest: memoryRequest,
				MemoryLimit:   memoryLimit,
			}

			// Check that at least one scaling option is specified
			if !opts.HasReplicaChange() && !opts.HasResourceChange() {
				return fmt.Errorf("at least one of --replicas, --cpu-request, --cpu-limit, --memory-request, or --memory-limit must be specified")
			}

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			mgr := manager.NewManager(profile)
			start := time.Now()
			scaleArgs := []string{fmt.Sprintf("--component=%s", component)}
			if opts.HasReplicaChange() {
				scaleArgs = append(scaleArgs, fmt.Sprintf("--replicas=%d", replicas))
			}
			scaleErr := mgr.Scale(ctx, instanceName, component, opts)
			auditLog(instanceName, "scale", scaleArgs, scaleErr, time.Since(start))
			return scaleErr
		},
	}

	cmd.Flags().StringVarP(&component, "component", "c", "", "Component to scale (required)")
	cmd.Flags().IntVarP(&replicas, "replicas", "r", 0, "Number of replicas (0 means no change)")
	cmd.Flags().StringVar(&cpuRequest, "cpu-request", "", "CPU request (e.g., '2', '500m')")
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", "CPU limit (e.g., '4', '1000m')")
	cmd.Flags().StringVar(&memoryRequest, "memory-request", "", "Memory request (e.g., '4Gi', '512Mi')")
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Memory limit (e.g., '8Gi', '1024Mi')")
	_ = cmd.MarkFlagRequired("component")

	return cmd
}

func newInstanceReplicasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replicas <instance-name>",
		Short: "Show current replica counts",
		Long: `Show the current replica count for each component in the instance.

For Kubernetes deployments, this shows actual running pod counts.
For local deployments, this shows standalone replica count (always 1 when running).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := manager.NewManager(profile)

			replicas, err := mgr.GetReplicas(ctx, instanceName)
			if err != nil {
				return err
			}

			fmt.Printf("Instance: %s\n", color.CyanString(instanceName))
			fmt.Println("Replicas:")

			// Order components for consistent output
			components := []string{"standalone", "proxy", "rootcoord", "querycoord", "datacoord", "indexcoord", "querynode", "datanode", "indexnode"}
			for _, comp := range components {
				if count, ok := replicas[comp]; ok {
					fmt.Printf("  %-12s %d\n", comp+":", count)
				}
			}

			return nil
		},
	}
	return cmd
}

func newInstanceUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade <instance-name> <version>",
		Short: "Upgrade Milvus to a new version",
		Long: `Upgrade the Milvus instance to a specified version.

For Kubernetes deployments, this triggers a rolling update managed by the Milvus Operator.
For local deployments, this pulls the new image and recreates the containers.

The upgrade process:
  1. Updates the Milvus image version in the deployment
  2. Performs a rolling update (Kubernetes) or container restart (local)
  3. Waits for the cluster to become healthy

Examples:
  # Upgrade to a specific version
  miup instance upgrade prod v2.5.5
  miup instance upgrade prod 2.5.5

  # Show current version before upgrading
  miup instance display prod`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			version := args[1]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			mgr := manager.NewManager(profile)
			start := time.Now()
			upgradeErr := mgr.Upgrade(ctx, instanceName, version)
			auditLog(instanceName, "upgrade", []string{version}, upgradeErr, time.Since(start))
			return upgradeErr
		},
	}
	return cmd
}

func newInstanceDestroyCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "destroy <instance-name>",
		Short: "Destroy an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mgr := manager.NewManager(profile)
			start := time.Now()
			destroyErr := mgr.Destroy(ctx, instanceName, force)
			auditLog(instanceName, "destroy", nil, destroyErr, time.Since(start))
			return destroyErr
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force destroy without confirmation")

	return cmd
}

func newInstanceLogsCmd() *cobra.Command {
	var (
		service string
		tail    int
	)

	cmd := &cobra.Command{
		Use:   "logs <instance-name>",
		Short: "Show instance logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := manager.NewManager(profile)

			logs, err := mgr.Logs(ctx, instanceName, service, tail)
			if err != nil {
				return err
			}

			fmt.Print(logs)
			return nil
		},
	}

	cmd.Flags().StringVarP(&service, "service", "s", "", "Service name (e.g., standalone, etcd, minio)")
	cmd.Flags().IntVarP(&tail, "tail", "n", 100, "Number of lines to show")

	return cmd
}

func newInstanceTemplateCmd() *cobra.Command {
	var (
		mode    string
		withTLS bool
	)

	cmd := &cobra.Command{
		Use:   "template",
		Short: "Print instance topology template",
		Long: `Print a topology template for deploying Milvus instances on Kubernetes.

Examples:
  miup instance template                    Standalone template
  miup instance template --tls              Standalone with TLS
  miup instance template --mode distributed Distributed template`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if withTLS {
				fmt.Print(kubernetesTLSTemplate)
			} else if mode == "distributed" {
				fmt.Print(kubernetesDistributedTemplate)
			} else {
				fmt.Print(kubernetesStandaloneTemplate)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "standalone", "Deployment mode: standalone or distributed")
	cmd.Flags().BoolVar(&withTLS, "tls", false, "Include TLS configuration in template")

	return cmd
}

func formatClusterStatus(status spec.ClusterStatus) string {
	switch status {
	case spec.StatusRunning:
		return color.GreenString("running")
	case spec.StatusStopped:
		return color.YellowString("stopped")
	case spec.StatusDeploying:
		return color.CyanString("deploying")
	case spec.StatusUpgrading:
		return color.CyanString("upgrading")
	default:
		return color.RedString("unknown")
	}
}

const kubernetesStandaloneTemplate = `# MiUp Kubernetes Topology - Standalone Mode
# Deploy with: miup instance deploy <instance-name> <this-file>
# Requires: Milvus Operator installed in your Kubernetes cluster

global:
  namespace: "milvus"
  storage_class: "standard"

milvus_servers:
  - host: 127.0.0.1
    port: 19530
    mode: standalone

# In-cluster etcd (managed by Milvus Operator)
etcd_servers:
  - host: 127.0.0.1
    client_port: 2379

# In-cluster MinIO (managed by Milvus Operator)
minio_servers:
  - host: 127.0.0.1
    port: 9000
    access_key: "minioadmin"
    secret_key: "minioadmin"
`

const kubernetesDistributedTemplate = `# MiUp Kubernetes Topology - Distributed Mode
# Deploy with: miup instance deploy <instance-name> <this-file>
# Requires: Milvus Operator installed in your Kubernetes cluster

global:
  namespace: "milvus"
  storage_class: "standard"

milvus_servers:
  - host: 127.0.0.1
    port: 19530
    mode: distributed
    components:
      proxy:
        replicas: 2
        resources:
          cpu: "1"
          memory: "2Gi"
      rootCoord:
        replicas: 1
      queryCoord:
        replicas: 1
      dataCoord:
        replicas: 1
      indexCoord:
        replicas: 1
      queryNode:
        replicas: 2
        resources:
          cpu: "2"
          memory: "4Gi"
      dataNode:
        replicas: 2
        resources:
          cpu: "1"
          memory: "2Gi"
      indexNode:
        replicas: 1
        resources:
          cpu: "2"
          memory: "4Gi"

# In-cluster etcd (managed by Milvus Operator)
etcd_servers:
  - host: 127.0.0.1
    client_port: 2379

# In-cluster MinIO (managed by Milvus Operator)
minio_servers:
  - host: 127.0.0.1
    port: 9000
    access_key: "minioadmin"
    secret_key: "minioadmin"

# External etcd example (uncomment to use):
# etcd_servers:
#   - host: etcd-cluster.etcd-system.svc.cluster.local
#     client_port: 2379

# External S3/MinIO example (uncomment to use):
# minio_servers:
#   - host: minio.minio-system.svc.cluster.local
#     port: 9000
#     access_key: "your-access-key"
#     secret_key: "your-secret-key"
`

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for miup.

To load completions:

Bash:
  # Linux:
  $ miup completion bash > /etc/bash_completion.d/miup
  # macOS:
  $ miup completion bash > $(brew --prefix)/etc/bash_completion.d/miup

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  # Linux:
  $ miup completion zsh > "${fpath[1]}/_miup"
  # macOS:
  $ miup completion zsh > $(brew --prefix)/share/zsh/site-functions/_miup

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ miup completion fish > ~/.config/fish/completions/miup.fish

PowerShell:
  PS> miup completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> miup completion powershell > miup.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unknown shell: %s", args[0])
			}
		},
	}
	return cmd
}

func newMirrorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Manage offline mirror for air-gapped environments",
		Long: `Mirror provides commands for managing Docker images for offline/air-gapped deployments.

This allows you to:
  - Pull all required images for Milvus deployment
  - Save images to a tar archive for transfer
  - Load images from a tar archive
  - Push images to a private registry

Examples:
  miup mirror pull                    Pull all required images
  miup mirror save -o milvus.tar      Save images to tar file
  miup mirror load -i milvus.tar      Load images from tar file
  miup mirror push registry.local     Push images to private registry`,
	}

	cmd.AddCommand(newMirrorPullCmd())
	cmd.AddCommand(newMirrorSaveCmd())
	cmd.AddCommand(newMirrorLoadCmd())
	cmd.AddCommand(newMirrorPushCmd())
	cmd.AddCommand(newMirrorListCmd())

	return cmd
}

func newMirrorPullCmd() *cobra.Command {
	var (
		milvusVersion string
		all           bool
		registry      string
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull Docker images for offline deployment",
		Long: `Pull all required Docker images for Milvus deployment.

This command pulls the following images:
  - milvusdb/milvus (Milvus server)
  - quay.io/coreos/etcd (etcd)
  - minio/minio (MinIO object storage)
  - prom/prometheus (optional, for monitoring)
  - grafana/grafana (optional, for monitoring)

Examples:
  miup mirror pull                                    Pull from public registries
  miup mirror pull --registry harbor.milvus.io       Pull from internal Harbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			images := getMilvusImages(milvusVersion, all, registry)

			for _, img := range images {
				logger.Info("Pulling image: %s", img)
				if err := pullImage(img); err != nil {
					return fmt.Errorf("failed to pull %s: %w", img, err)
				}
				logger.Success("Pulled: %s", img)
			}

			logger.Success("All images pulled successfully!")
			return nil
		},
	}

	cmd.Flags().StringVar(&milvusVersion, "milvus.version", "v2.5.4", "Milvus version")
	cmd.Flags().BoolVar(&all, "all", false, "Include monitoring images (Prometheus, Grafana)")
	cmd.Flags().StringVar(&registry, "registry", "", "Private registry address (e.g., harbor.milvus.io)")

	return cmd
}

func newMirrorSaveCmd() *cobra.Command {
	var (
		output        string
		milvusVersion string
		all           bool
		registry      string
	)

	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save Docker images to a tar archive",
		Long: `Save all required Docker images to a tar archive for offline transfer.

The tar archive can be transferred to air-gapped environments and loaded using:
  miup mirror load -i <archive.tar>

Examples:
  miup mirror save -o milvus.tar                           Save from public registries
  miup mirror save -o milvus.tar --registry harbor.milvus.io  Save from internal Harbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output == "" {
				output = fmt.Sprintf("milvus-images-%s.tar", milvusVersion)
			}

			images := getMilvusImages(milvusVersion, all, registry)

			logger.Info("Saving %d images to %s...", len(images), output)
			if err := saveImages(images, output); err != nil {
				return fmt.Errorf("failed to save images: %w", err)
			}

			logger.Success("Images saved to: %s", output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output tar file (default: milvus-images-<version>.tar)")
	cmd.Flags().StringVar(&milvusVersion, "milvus.version", "v2.5.4", "Milvus version")
	cmd.Flags().BoolVar(&all, "all", false, "Include monitoring images (Prometheus, Grafana)")
	cmd.Flags().StringVar(&registry, "registry", "", "Private registry address (e.g., harbor.milvus.io)")

	return cmd
}

func newMirrorLoadCmd() *cobra.Command {
	var input string

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load Docker images from a tar archive",
		Long: `Load Docker images from a tar archive created by 'miup mirror save'.

This is typically used in air-gapped environments after transferring the tar archive.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("input file is required (-i)")
			}

			logger.Info("Loading images from %s...", input)
			if err := loadImages(input); err != nil {
				return fmt.Errorf("failed to load images: %w", err)
			}

			logger.Success("Images loaded successfully!")
			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input tar file (required)")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func newMirrorPushCmd() *cobra.Command {
	var (
		milvusVersion  string
		all            bool
		sourceRegistry string
	)

	cmd := &cobra.Command{
		Use:   "push <registry>",
		Short: "Push images to a private registry",
		Long: `Push all Milvus images to a private Docker registry.

This re-tags and pushes images to your private registry for use in air-gapped environments.

Examples:
  miup mirror push registry.local:5000
  miup mirror push harbor.example.com/milvus
  miup mirror push registry.local:5000 --source-registry harbor.milvus.io`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetRegistry := args[0]
			images := getMilvusImages(milvusVersion, all, sourceRegistry)

			for _, img := range images {
				newTag := retagImage(img, targetRegistry)
				logger.Info("Pushing %s -> %s", img, newTag)

				if err := tagAndPushImage(img, newTag); err != nil {
					return fmt.Errorf("failed to push %s: %w", newTag, err)
				}
				logger.Success("Pushed: %s", newTag)
			}

			logger.Success("All images pushed to %s", targetRegistry)
			return nil
		},
	}

	cmd.Flags().StringVar(&milvusVersion, "milvus.version", "v2.5.4", "Milvus version")
	cmd.Flags().BoolVar(&all, "all", false, "Include monitoring images (Prometheus, Grafana)")
	cmd.Flags().StringVar(&sourceRegistry, "source-registry", "", "Source registry to pull images from (e.g., harbor.milvus.io)")

	return cmd
}

func newMirrorListCmd() *cobra.Command {
	var (
		milvusVersion string
		all           bool
		registry      string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List required Docker images",
		Long: `List all Docker images required for Milvus deployment.

Examples:
  miup mirror list                               List images from public registries
  miup mirror list --registry harbor.milvus.io  List images from internal Harbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			images := getMilvusImages(milvusVersion, all, registry)

			fmt.Println("Required images for Milvus deployment:")
			for _, img := range images {
				fmt.Printf("  - %s\n", img)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&milvusVersion, "milvus.version", "v2.5.4", "Milvus version")
	cmd.Flags().BoolVar(&all, "all", false, "Include monitoring images (Prometheus, Grafana)")
	cmd.Flags().StringVar(&registry, "registry", "", "Private registry address (e.g., harbor.milvus.io)")

	return cmd
}

// getMilvusImages returns the list of Docker images required for Milvus deployment
// If registry is provided, images will be prefixed with the registry address
func getMilvusImages(milvusVersion string, includeMonitoring bool, registry string) []string {
	var images []string

	if registry != "" {
		// Use internal registry (e.g., harbor.milvus.io)
		// Format: registry/project/image:tag
		images = []string{
			fmt.Sprintf("%s/milvus/milvus:%s", registry, milvusVersion),
			fmt.Sprintf("%s/milvus-ci/etcd:3.5.18-r0", registry),
			fmt.Sprintf("%s/milvus-ci/minio:RELEASE.2023-03-20T20-16-18Z", registry),
		}
		if includeMonitoring {
			images = append(images,
				fmt.Sprintf("%s/milvus-ci/prometheus:latest", registry),
				fmt.Sprintf("%s/milvus-ci/grafana:latest", registry),
			)
		}
	} else {
		// Use public registries
		images = []string{
			fmt.Sprintf("milvusdb/milvus:%s", milvusVersion),
			"quay.io/coreos/etcd:v3.5.18",
			"minio/minio:RELEASE.2023-03-20T20-16-18Z",
		}
		if includeMonitoring {
			images = append(images,
				"prom/prometheus:latest",
				"grafana/grafana:latest",
			)
		}
	}

	return images
}

// pullImage pulls a Docker image
func pullImage(image string) error {
	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// saveImages saves Docker images to a tar file
func saveImages(images []string, output string) error {
	args := append([]string{"save", "-o", output}, images...)
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// loadImages loads Docker images from a tar file
func loadImages(input string) error {
	cmd := exec.Command("docker", "load", "-i", input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// retagImage generates a new tag for pushing to a private registry
func retagImage(image, registry string) string {
	// Extract image name without registry
	parts := strings.Split(image, "/")
	var imageName string
	if len(parts) == 1 {
		imageName = parts[0]
	} else {
		imageName = strings.Join(parts[len(parts)-2:], "/")
	}
	return fmt.Sprintf("%s/%s", registry, imageName)
}

// tagAndPushImage tags and pushes an image to a registry
func tagAndPushImage(source, target string) error {
	// Tag the image
	tagCmd := exec.Command("docker", "tag", source, target)
	if err := tagCmd.Run(); err != nil {
		return fmt.Errorf("failed to tag: %w", err)
	}

	// Push the image
	pushCmd := exec.Command("docker", "push", target)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	return pushCmd.Run()
}

const kubernetesTLSTemplate = `# MiUp Kubernetes Topology - Standalone Mode with TLS
# Deploy with: miup instance deploy <instance-name> <this-file>
# Requires: Milvus Operator installed in your Kubernetes cluster
#
# Before deploying, create TLS secret:
#   kubectl create secret generic milvus-tls \
#     --from-file=server.pem --from-file=server.key --from-file=ca.pem \
#     -n milvus

global:
  namespace: "milvus"
  storage_class: "standard"
  tls:
    enabled: true
    mode: 1  # 1 = one-way TLS, 2 = two-way TLS (mutual TLS)
    secret_name: "milvus-tls"  # K8s secret containing TLS certificates
    # internal_enabled: false  # Enable TLS for internal component communication

milvus_servers:
  - host: 127.0.0.1
    port: 19530
    mode: standalone

# In-cluster etcd (managed by Milvus Operator)
etcd_servers:
  - host: 127.0.0.1
    client_port: 2379

# In-cluster MinIO (managed by Milvus Operator)
minio_servers:
  - host: 127.0.0.1
    port: 9000
    access_key: "minioadmin"
    secret_key: "minioadmin"
`

// ==================== Bench Commands ====================
// Bench commands wrap go-vdbbench for Milvus benchmarking
// Similar to how TiUP bench wraps go-tpc

func newBenchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Run benchmark tests on Milvus",
		Long: `Benchmark tools for testing Milvus performance using go-vdbbench.

go-vdbbench is a pure Go vector database benchmark tool (similar to go-tpc for TiDB).
It provides high-performance benchmarking for Milvus without external dependencies.

Commands:
  milvus    Run benchmark against Milvus

Examples:
  miup bench milvus prepare --uri localhost:19530              # Prepare test data
  miup bench milvus search --uri localhost:19530               # Run search benchmark
  miup bench milvus insert --uri localhost:19530               # Run insert benchmark
  miup bench milvus cleanup --uri localhost:19530              # Clean up test data`,
	}

	cmd.AddCommand(newBenchMilvusCmd())

	return cmd
}

// newBenchMilvusCmd creates Milvus benchmark commands
func newBenchMilvusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "milvus",
		Short: "Benchmark Milvus vector database",
		Long: `Run benchmark tests against a Milvus instance.

Available commands:
  prepare   Prepare test data (create collection, insert data, build index)
  search    Run search performance test
  insert    Run insert performance test
  cleanup   Clean up test data`,
	}

	cmd.AddCommand(newBenchMilvusPrepareCmd())
	cmd.AddCommand(newBenchMilvusSearchCmd())
	cmd.AddCommand(newBenchMilvusInsertCmd())
	cmd.AddCommand(newBenchMilvusCleanupCmd())

	return cmd
}

// benchFlags holds common benchmark flags
type benchFlags struct {
	uri         string
	username    string
	password    string
	dbName      string
	collection  string
	datasetName string
	dimension   int
	dataSize    int
	threads     int
	duration    int
	batchSize   int
	topK        int
	indexType   string
}

func addBenchFlags(cmd *cobra.Command, flags *benchFlags) {
	cmd.Flags().StringVar(&flags.uri, "uri", "localhost:19530", "Milvus server URI")
	cmd.Flags().StringVar(&flags.username, "username", "", "Username for authentication")
	cmd.Flags().StringVar(&flags.password, "password", "", "Password for authentication")
	cmd.Flags().StringVar(&flags.dbName, "db", "", "Database name")
	cmd.Flags().StringVar(&flags.collection, "collection", "benchmark_collection", "Collection name")
	cmd.Flags().StringVar(&flags.datasetName, "dataset", "small", "Dataset name (small, medium, large, cohere-100k, cohere-1m, openai-50k)")
	cmd.Flags().IntVar(&flags.dimension, "dimension", 0, "Vector dimension (overrides dataset default)")
	cmd.Flags().IntVar(&flags.dataSize, "size", 0, "Data size (overrides dataset default)")
	cmd.Flags().IntVar(&flags.threads, "threads", 10, "Number of concurrent threads")
	cmd.Flags().IntVar(&flags.duration, "duration", 60, "Test duration in seconds")
	cmd.Flags().IntVar(&flags.batchSize, "batch-size", 1000, "Batch size for insert")
	cmd.Flags().IntVar(&flags.topK, "top-k", 10, "Number of results for search")
	cmd.Flags().StringVar(&flags.indexType, "index-type", "IVF_FLAT", "Index type (FLAT, IVF_FLAT, HNSW)")
}

func buildVdbbenchArgs(subcmd string, flags *benchFlags) []string {
	args := []string{"milvus", subcmd}
	args = append(args, "--uri", flags.uri)
	if flags.username != "" {
		args = append(args, "--username", flags.username)
	}
	if flags.password != "" {
		args = append(args, "--password", flags.password)
	}
	if flags.dbName != "" {
		args = append(args, "--db", flags.dbName)
	}
	args = append(args, "--collection", flags.collection)
	args = append(args, "--dataset", flags.datasetName)
	if flags.dimension > 0 {
		args = append(args, "--dimension", fmt.Sprintf("%d", flags.dimension))
	}
	if flags.dataSize > 0 {
		args = append(args, "--size", fmt.Sprintf("%d", flags.dataSize))
	}
	args = append(args, "--threads", fmt.Sprintf("%d", flags.threads))
	args = append(args, "--duration", fmt.Sprintf("%d", flags.duration))
	args = append(args, "--batch-size", fmt.Sprintf("%d", flags.batchSize))
	args = append(args, "--top-k", fmt.Sprintf("%d", flags.topK))
	args = append(args, "--index-type", flags.indexType)
	return args
}

func runGoVdbbench(args []string) error {
	// Try to find go-vdbbench binary
	vdbbenchPath := findVdbbenchBinary()
	if vdbbenchPath == "" {
		return fmt.Errorf("go-vdbbench not found. Please build it first:\n  cd tools/go-vdbbench && go build -o go-vdbbench ./cmd/go-vdbbench")
	}

	logger.Debug("Running: %s %v", vdbbenchPath, args)

	cmd := exec.Command(vdbbenchPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func findVdbbenchBinary() string {
	// Check common locations
	locations := []string{
		"./go-vdbbench",
		"./tools/go-vdbbench/go-vdbbench",
		"go-vdbbench",
	}

	// Check if MIUP_HOME is set
	if home := os.Getenv("MIUP_HOME"); home != "" {
		locations = append([]string{
			home + "/bin/go-vdbbench",
			home + "/tools/go-vdbbench/go-vdbbench",
		}, locations...)
	}

	// Get executable path for relative paths
	if execPath, err := os.Executable(); err == nil {
		execDir := strings.TrimSuffix(execPath, "/miup")
		locations = append([]string{
			execDir + "/go-vdbbench",
			execDir + "/../tools/go-vdbbench/go-vdbbench",
		}, locations...)
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	// Try PATH
	if path, err := exec.LookPath("go-vdbbench"); err == nil {
		return path
	}

	return ""
}

func newBenchMilvusPrepareCmd() *cobra.Command {
	var flags benchFlags

	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare test data",
		Long: `Prepare test data for benchmarking.

This command will:
  1. Create a new collection
  2. Insert test vectors
  3. Build index
  4. Load collection into memory

Available datasets:
  small       10,000 vectors (128 dim)
  medium      100,000 vectors (128 dim)
  large       1,000,000 vectors (128 dim)
  cohere-100k 100,000 vectors (768 dim)
  cohere-1m   1,000,000 vectors (768 dim)
  openai-50k  50,000 vectors (1536 dim)
  openai-500k 500,000 vectors (1536 dim)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vdbbenchArgs := buildVdbbenchArgs("prepare", &flags)
			return runGoVdbbench(vdbbenchArgs)
		},
	}

	addBenchFlags(cmd, &flags)
	return cmd
}

func newBenchMilvusSearchCmd() *cobra.Command {
	var flags benchFlags

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run search performance test",
		Long: `Run search performance test against Milvus.

The test will execute concurrent vector similarity searches and measure:
  - QPS (queries per second)
  - Latency (avg, p50, p95, p99)
  - Error rate

Note: Requires data to be prepared first using 'miup bench milvus prepare'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vdbbenchArgs := buildVdbbenchArgs("search", &flags)
			return runGoVdbbench(vdbbenchArgs)
		},
	}

	addBenchFlags(cmd, &flags)
	return cmd
}

func newBenchMilvusInsertCmd() *cobra.Command {
	var flags benchFlags

	cmd := &cobra.Command{
		Use:   "insert",
		Short: "Run insert performance test",
		Long: `Run insert performance test against Milvus.

The test will execute concurrent batch inserts and measure:
  - Throughput (batches per second)
  - Latency (avg, p50, p95, p99)
  - Error rate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vdbbenchArgs := buildVdbbenchArgs("insert", &flags)
			return runGoVdbbench(vdbbenchArgs)
		},
	}

	addBenchFlags(cmd, &flags)
	return cmd
}

func newBenchMilvusCleanupCmd() *cobra.Command {
	var flags benchFlags

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up test data",
		Long:  `Remove the benchmark collection and all test data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vdbbenchArgs := buildVdbbenchArgs("cleanup", &flags)
			return runGoVdbbench(vdbbenchArgs)
		},
	}

	addBenchFlags(cmd, &flags)
	return cmd
}

func newInstanceConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage instance configuration",
		Long: `Manage Milvus configuration for an instance.

For Kubernetes deployments, configuration is stored in the Milvus CRD spec.config field.
For local deployments, configuration is stored in the user.yaml file.

Subcommands:
  show    Show current configuration
  set     Set configuration values
  import  Import configuration from a YAML file
  export  Export configuration to stdout (YAML format)

Examples:
  miup instance config show prod
  miup instance config set prod common.security.tlsMode=1
  miup instance config import prod config.yaml
  miup instance config export prod > config.yaml`,
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigImportCmd())
	cmd.AddCommand(newConfigExportCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var key string

	cmd := &cobra.Command{
		Use:   "show <instance-name>",
		Short: "Show current configuration",
		Long: `Show the current Milvus configuration for an instance.

Use --key to show a specific configuration section.

Examples:
  miup instance config show prod
  miup instance config show prod --key common
  miup instance config show prod --key proxy`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := manager.NewManager(profile)

			config, err := mgr.GetConfig(ctx, instanceName)
			if err != nil {
				return err
			}

			// Filter by key if specified
			if key != "" {
				if val, ok := config[key]; ok {
					config = map[string]interface{}{key: val}
				} else {
					return fmt.Errorf("configuration key '%s' not found", key)
				}
			}

			if len(config) == 0 {
				fmt.Println("No configuration set.")
				return nil
			}

			// Output as YAML
			data, err := yaml.Marshal(config)
			if err != nil {
				return fmt.Errorf("failed to format config: %w", err)
			}

			fmt.Printf("Instance: %s\n", color.CyanString(instanceName))
			fmt.Println("Configuration:")
			fmt.Println(string(data))

			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Show only the specified configuration key")

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <instance-name> <key=value>...",
		Short: "Set configuration values",
		Long: `Set one or more configuration values for an instance.

Configuration keys use dot notation for nested values.
After setting, the instance will be restarted to apply changes.

Examples:
  miup instance config set prod common.security.tlsMode=1
  miup instance config set prod proxy.maxTaskNum=1024
  miup instance config set prod queryNode.gracefulTime=5000`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			keyValues := args[1:]

			// Parse key=value pairs into nested config
			config := make(map[string]interface{})
			for _, kv := range keyValues {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid format '%s': expected key=value", kv)
				}
				key, value := parts[0], parts[1]

				// Parse the value (try number, bool, then string)
				var parsedValue any = value
				var intVal int
				if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
					parsedValue = intVal
				} else if value == "true" {
					parsedValue = true
				} else if value == "false" {
					parsedValue = false
				}

				// Build nested structure from dot notation
				setNestedValue(config, key, parsedValue)
			}

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			mgr := manager.NewManager(profile)
			return mgr.SetConfig(ctx, instanceName, config)
		},
	}

	return cmd
}

// setNestedValue sets a value in a nested map using dot notation key
func setNestedValue(m map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, ok := current[part]; !ok {
				current[part] = make(map[string]interface{})
			}
			current = current[part].(map[string]interface{})
		}
	}
}

func newConfigImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <instance-name> <config-file>",
		Short: "Import configuration from a YAML file",
		Long: `Import Milvus configuration from a YAML file.

The configuration will be merged with existing configuration.
After importing, the instance will be restarted to apply changes.

Examples:
  miup instance config import prod config.yaml
  miup instance config import prod /path/to/milvus.yaml`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			configFile := args[1]

			// Read config file
			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}

			var config map[string]interface{}
			if err := yaml.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			mgr := manager.NewManager(profile)
			return mgr.SetConfig(ctx, instanceName, config)
		},
	}

	return cmd
}

func newConfigExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <instance-name>",
		Short: "Export configuration to stdout",
		Long: `Export the current Milvus configuration to stdout in YAML format.

You can redirect the output to a file for backup or modification.

Examples:
  miup instance config export prod
  miup instance config export prod > config.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := manager.NewManager(profile)

			config, err := mgr.GetConfig(ctx, instanceName)
			if err != nil {
				return err
			}

			if len(config) == 0 {
				fmt.Println("# No configuration set")
				return nil
			}

			// Output as YAML
			data, err := yaml.Marshal(config)
			if err != nil {
				return fmt.Errorf("failed to format config: %w", err)
			}

			fmt.Print(string(data))

			return nil
		},
	}

	return cmd
}

func newInstanceDiagnoseCmd() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "diagnose <instance-name>",
		Short: "Run health diagnostics on an instance",
		Long: `Perform comprehensive health diagnostics on a Milvus instance.

This command checks:
  - Component health status (standalone/proxy/querynode/datanode/etc.)
  - Service connectivity (Milvus, etcd, MinIO endpoints)
  - Resource usage and limits
  - Common issues and provides suggestions

For Kubernetes deployments, it inspects the Milvus CRD status and conditions.
For local deployments, it checks Docker container health.

Examples:
  miup instance diagnose prod
  miup instance diagnose prod --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			ctx := context.Background()
			mgr := manager.NewManager(profile)

			result, err := mgr.Diagnose(ctx, instanceName)
			if err != nil {
				return err
			}

			if outputJSON {
				return printDiagnoseJSON(result)
			}

			return printDiagnoseResult(instanceName, result)
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	return cmd
}

func printDiagnoseResult(instanceName string, result *executor.DiagnoseResult) error {
	// Header
	fmt.Printf("Instance: %s\n", color.CyanString(instanceName))
	fmt.Println(strings.Repeat("-", 50))

	// Summary
	if result.Healthy {
		fmt.Printf("Status: %s\n", color.GreenString("HEALTHY"))
	} else {
		fmt.Printf("Status: %s\n", color.RedString("UNHEALTHY"))
	}
	fmt.Printf("Summary: %s\n\n", result.Summary)

	// Components
	fmt.Println(color.CyanString("Components:"))
	for _, comp := range result.Components {
		statusIcon := getStatusIcon(comp.Status)
		if comp.Replicas > 0 {
			fmt.Printf("  %s %-12s %s (%d/%d replicas)\n", statusIcon, comp.Name, comp.Message, comp.Ready, comp.Replicas)
		} else {
			fmt.Printf("  %s %-12s %s\n", statusIcon, comp.Name, comp.Message)
		}
	}
	fmt.Println()

	// Connectivity
	if len(result.Connectivity) > 0 {
		fmt.Println(color.CyanString("Connectivity:"))
		for _, conn := range result.Connectivity {
			statusIcon := getStatusIcon(conn.Status)
			fmt.Printf("  %s %-15s %s - %s\n", statusIcon, conn.Name, conn.Target, conn.Message)
		}
		fmt.Println()
	}

	// Resources
	if len(result.Resources) > 0 {
		fmt.Println(color.CyanString("Resources:"))
		for _, res := range result.Resources {
			statusIcon := getStatusIcon(res.Status)
			fmt.Printf("  %s %-15s %s (limit: %s) - %s\n", statusIcon, res.Name, res.Usage, res.Limit, res.Message)
		}
		fmt.Println()
	}

	// Issues
	if len(result.Issues) > 0 {
		fmt.Println(color.YellowString("Issues Found:"))
		for i, issue := range result.Issues {
			severityColor := color.YellowString
			if issue.Severity == executor.CheckStatusError {
				severityColor = color.RedString
			}
			fmt.Printf("  %d. [%s] %s\n", i+1, severityColor(string(issue.Severity)), issue.Description)
			fmt.Printf("     Component: %s\n", issue.Component)
			fmt.Printf("     Suggestion: %s\n", color.CyanString(issue.Suggestion))
		}
	} else {
		fmt.Println(color.GreenString("No issues found."))
	}

	return nil
}

func getStatusIcon(status executor.CheckStatus) string {
	switch status {
	case executor.CheckStatusOK:
		return color.GreenString("[OK]")
	case executor.CheckStatusWarning:
		return color.YellowString("[WARN]")
	case executor.CheckStatusError:
		return color.RedString("[ERR]")
	default:
		return "[?]"
	}
}

func printDiagnoseJSON(result *executor.DiagnoseResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func newInstanceCheckCmd() *cobra.Command {
	var (
		kubeconfig   string
		kubeContext  string
		namespace    string
		storageClass string
		outputJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check environment before deployment",
		Long: `Perform pre-deployment environment checks for Kubernetes deployment.

This command verifies:
  - Kubernetes cluster connectivity
  - Kubernetes version compatibility (requires 1.20+)
  - Milvus Operator installation status
  - Target namespace existence
  - Storage class availability
  - Resource quota capacity

Run this check before deploying a Milvus instance to ensure the environment is ready.

Examples:
  miup instance check
  miup instance check --kubeconfig ~/.kube/config
  miup instance check --namespace milvus --storage-class standard
  miup instance check --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker, err := check.NewChecker(check.Options{
				Kubeconfig:   kubeconfig,
				Context:      kubeContext,
				Namespace:    namespace,
				StorageClass: storageClass,
			})
			if err != nil {
				return err
			}

			ctx := context.Background()
			report, err := checker.Run(ctx)
			if err != nil {
				return err
			}

			if outputJSON {
				return printCheckJSON(report)
			}

			return printCheckReport(report)
		},
	}

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	cmd.Flags().StringVar(&kubeContext, "context", "", "Kubernetes context to use")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "milvus", "Target namespace for deployment")
	cmd.Flags().StringVar(&storageClass, "storage-class", "", "Storage class to verify")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	return cmd
}

func printCheckReport(report *check.Report) error {
	// Header
	fmt.Println(color.CyanString("Kubernetes Environment Check"))
	fmt.Println(strings.Repeat("-", 50))

	// Results
	for _, r := range report.Results {
		var statusIcon string
		switch r.Status {
		case check.StatusPass:
			statusIcon = color.GreenString("[PASS]")
		case check.StatusWarn:
			statusIcon = color.YellowString("[WARN]")
		case check.StatusFail:
			statusIcon = color.RedString("[FAIL]")
		}

		fmt.Printf("  %s %s\n", statusIcon, r.Name)
		fmt.Printf("       %s\n", r.Message)
		if r.Suggest != "" {
			fmt.Printf("       %s %s\n", color.CyanString("Suggestion:"), r.Suggest)
		}
	}

	// Summary
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Summary: %d passed, %d warnings, %d failed\n",
		report.Summary.Passed, report.Summary.Warned, report.Summary.Failed)

	if report.CanDeploy {
		fmt.Println(color.GreenString("Environment is ready for deployment!"))
	} else {
		fmt.Println(color.RedString("Environment is NOT ready. Please fix the failed checks."))
		return fmt.Errorf("environment check failed")
	}

	return nil
}

func printCheckJSON(report *check.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func newInstanceAuditCmd() *cobra.Command {
	var (
		instance   string
		command    string
		limit      int
		outputJSON bool
		clear      bool
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View operation audit logs",
		Long: `View audit logs of instance management operations.

The audit log records all instance operations including:
  - deploy, start, stop, destroy
  - scale, upgrade
  - config changes
  - diagnose

Each entry includes timestamp, user, command, status, and duration.

Examples:
  miup instance audit                          Show last 20 audit entries
  miup instance audit --limit 50               Show last 50 entries
  miup instance audit --instance prod          Filter by instance name
  miup instance audit --command deploy         Filter by command
  miup instance audit --json                   Output in JSON format
  miup instance audit --clear                  Clear audit logs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := audit.NewLogger()
			if err != nil {
				return fmt.Errorf("failed to initialize audit logger: %w", err)
			}

			if clear {
				if err := logger.Clear(); err != nil {
					return fmt.Errorf("failed to clear audit logs: %w", err)
				}
				fmt.Println("Audit logs cleared.")
				return nil
			}

			// Set default limit
			if limit <= 0 {
				limit = 20
			}

			entries, err := logger.Query(audit.QueryOptions{
				Instance: instance,
				Command:  command,
				Limit:    limit,
			})
			if err != nil {
				return fmt.Errorf("failed to query audit logs: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("No audit logs found.")
				return nil
			}

			if outputJSON {
				return printAuditJSON(entries)
			}

			return printAuditTable(entries)
		},
	}

	cmd.Flags().StringVarP(&instance, "instance", "i", "", "Filter by instance name")
	cmd.Flags().StringVarP(&command, "command", "c", "", "Filter by command")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Number of entries to show")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear all audit logs")

	return cmd
}

func printAuditTable(entries []audit.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tINSTANCE\tCOMMAND\tSTATUS\tDURATION\tUSER")
	fmt.Fprintln(w, "---------\t--------\t-------\t------\t--------\t----")

	for _, e := range entries {
		var statusStr string
		switch e.Status {
		case audit.StatusSuccess:
			statusStr = color.GreenString("success")
		case audit.StatusFailed:
			statusStr = color.RedString("failed")
		case audit.StatusRunning:
			statusStr = color.YellowString("running")
		default:
			statusStr = string(e.Status)
		}

		instance := e.Instance
		if instance == "" {
			instance = "-"
		}

		duration := "-"
		if e.Duration > 0 {
			duration = e.Duration.Round(time.Millisecond).String()
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			instance,
			e.Command,
			statusStr,
			duration,
			e.User,
		)
	}

	w.Flush()

	// Show error details for failed entries
	for _, e := range entries {
		if e.Status == audit.StatusFailed && e.Error != "" {
			fmt.Printf("\n%s %s failed: %s\n",
				color.RedString("[ERROR]"),
				e.Command,
				e.Error,
			)
		}
	}

	return nil
}

func printAuditJSON(entries []audit.Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error: %v", err))
		os.Exit(1)
	}
}
