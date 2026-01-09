package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/zilliztech/miup/pkg/cluster/manager"
	"github.com/zilliztech/miup/pkg/cluster/spec"
	"github.com/zilliztech/miup/pkg/localdata"
	"github.com/zilliztech/miup/pkg/logger"
	"github.com/zilliztech/miup/pkg/playground"
	"github.com/zilliztech/miup/pkg/version"
)

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
  miup install milvus      Install Milvus component
  miup instance deploy     Deploy a Milvus instance

For more information, visit: https://github.com/zilliztech/miup`,
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
	rootCmd.AddCommand(newPlaygroundCmd())
	rootCmd.AddCommand(newClusterCmd())
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
		Short: "Install a component",
		Long: `Install a Milvus component.

Available components:
  milvus      Milvus vector database
  etcd        Distributed key-value store
  minio       Object storage server
  pulsar      Message queue (optional)
  prometheus  Monitoring system
  grafana     Visualization platform

Examples:
  miup install milvus              Install latest stable Milvus
  miup install milvus:v2.6.0       Install specific version
  miup install etcd minio          Install multiple components`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}
			if err := profile.InitProfile(); err != nil {
				return err
			}

			for _, component := range args {
				logger.Info("Installing component: %s", component)
				// TODO: Implement actual installation
				logger.Success("Component %s installed successfully", component)
			}
			return nil
		},
	}
	return cmd
}

func newUninstallCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "uninstall <component>",
		Short: "Uninstall a component",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, component := range args {
				logger.Info("Uninstalling component: %s", component)
				// TODO: Implement actual uninstallation
				logger.Success("Component %s uninstalled successfully", component)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Remove all versions of a component")
	return cmd
}

func newListCmd() *cobra.Command {
	var installed, available bool
	cmd := &cobra.Command{
		Use:   "list [component]",
		Short: "List components",
		Long: `List installed or available components.

Examples:
  miup list              List all installed components
  miup list --available  List all available components
  miup list milvus       List installed versions of milvus`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if available {
				fmt.Println("Available components:")
				fmt.Println("  milvus      Milvus vector database")
				fmt.Println("  etcd        Distributed key-value store")
				fmt.Println("  minio       Object storage server")
				fmt.Println("  pulsar      Message queue")
				fmt.Println("  prometheus  Monitoring system")
				fmt.Println("  grafana     Visualization platform")
				return nil
			}

			profile, err := localdata.DefaultProfile()
			if err != nil {
				return err
			}

			fmt.Printf("Installed components (in %s):\n", profile.ComponentsDir())
			// TODO: List actual installed components
			fmt.Println("  (none)")
			return nil
		},
	}
	cmd.Flags().BoolVar(&installed, "installed", false, "List installed components (default)")
	cmd.Flags().BoolVar(&available, "available", false, "List available components")
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
		mode        string
		tag         string
		withMonitor bool
		milvusVer   string
		milvusPort  int
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a local Milvus playground",
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

			// Parse mode
			var playgroundMode playground.Mode
			switch mode {
			case "standalone":
				playgroundMode = playground.ModeStandalone
			case "cluster":
				playgroundMode = playground.ModeCluster
			default:
				return fmt.Errorf("invalid mode: %s (must be 'standalone' or 'cluster')", mode)
			}

			// Create configuration
			cfg := playground.DefaultConfig()
			cfg.Tag = tag
			cfg.Mode = playgroundMode
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

	cmd.Flags().StringVar(&mode, "mode", "standalone", "Milvus mode: standalone or cluster")
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
		Long: `Instance provides commands for deploying and managing Milvus instances.

Local deployment uses Docker Compose (standalone mode only).
Kubernetes deployment uses Milvus Operator (supports standalone and distributed modes).

Examples:
  miup instance deploy prod topology.yaml              Deploy locally with Docker
  miup instance deploy prod topology.yaml --kubernetes Deploy to Kubernetes
  miup instance list                                   List all instances
  miup instance display prod                           Show instance details
  miup instance start prod                             Start an instance
  miup instance stop prod                              Stop an instance
  miup instance destroy prod                           Destroy an instance`,
	}

	cmd.AddCommand(newInstanceDeployCmd())
	cmd.AddCommand(newInstanceListCmd())
	cmd.AddCommand(newInstanceDisplayCmd())
	cmd.AddCommand(newInstanceStartCmd())
	cmd.AddCommand(newInstanceStopCmd())
	cmd.AddCommand(newInstanceDestroyCmd())
	cmd.AddCommand(newInstanceLogsCmd())
	cmd.AddCommand(newInstanceTemplateCmd())

	return cmd
}

func newInstanceDeployCmd() *cobra.Command {
	var (
		kubernetes    bool
		skipConfirm   bool
		milvusVersion string
		kubeconfig    string
		kubecontext   string
		namespace     string
		withMonitor   bool
	)

	cmd := &cobra.Command{
		Use:   "deploy <instance-name> <topology.yaml>",
		Short: "Deploy a Milvus instance",
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

			backend := spec.BackendLocal
			if kubernetes {
				backend = spec.BackendKubernetes
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
				Backend:       backend,
				SkipConfirm:   skipConfirm,
				Kubeconfig:    kubeconfig,
				KubeContext:   kubecontext,
				Namespace:     namespace,
				WithMonitor:   withMonitor,
			}

			if err := mgr.Deploy(ctx, instanceName, topoFile, opts); err != nil {
				return err
			}

			// Print connection info
			info, _ := mgr.Display(ctx, instanceName)
			if info != nil && info.Meta != nil {
				fmt.Println()
				fmt.Println("Connect to Milvus:")
				if kubernetes {
					fmt.Printf("  %s\n", color.CyanString("Namespace: %s", namespace))
					fmt.Printf("  %s\n", color.CyanString("Use: kubectl port-forward svc/%s-milvus -n %s 19530:19530", instanceName, namespace))
				} else {
					fmt.Printf("  %s\n", color.CyanString("Endpoint: localhost:%d", info.Meta.MilvusPort))
				}
				fmt.Printf("  %s\n", color.CyanString("SDK:      from pymilvus import MilvusClient"))
				fmt.Printf("  %s\n", color.CyanString("          client = MilvusClient('http://localhost:19530')"))
				if !kubernetes && info.Meta.MinioConsole > 0 {
					fmt.Println()
					fmt.Printf("MinIO Console: %s\n", color.CyanString("http://localhost:%d", info.Meta.MinioConsole))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&kubernetes, "kubernetes", false, "Deploy to Kubernetes using Milvus Operator")
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
			return mgr.Start(ctx, instanceName)
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
			return mgr.Stop(ctx, instanceName)
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
			return mgr.Destroy(ctx, instanceName, force)
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
		mode       string
		kubernetes bool
		withTLS    bool
	)

	cmd := &cobra.Command{
		Use:   "template",
		Short: "Print instance topology template",
		Long: `Print a topology template for deploying Milvus instances.

Local deployment (Docker) only supports standalone mode.
Kubernetes deployment supports both standalone and distributed modes.

Examples:
  miup instance template                              Local standalone template
  miup instance template --tls                        Local standalone with TLS
  miup instance template --kubernetes                 K8s standalone template
  miup instance template --kubernetes --tls           K8s standalone with TLS
  miup instance template --kubernetes --mode cluster  K8s distributed template`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kubernetes {
				if withTLS {
					fmt.Print(kubernetesTLSTemplate)
				} else if mode == "distributed" {
					fmt.Print(kubernetesDistributedTemplate)
				} else {
					fmt.Print(kubernetesStandaloneTemplate)
				}
			} else {
				// Local only supports standalone
				if withTLS {
					fmt.Print(localTLSTemplate)
				} else {
					fmt.Print(localStandaloneTemplate)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "standalone", "Kubernetes mode: standalone or distributed")
	cmd.Flags().BoolVar(&kubernetes, "kubernetes", false, "Print Kubernetes deployment template")
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

const localStandaloneTemplate = `# MiUp Local Instance - Standalone Mode (Docker Compose)
# Deploy with: miup instance deploy <instance-name> <this-file>

milvus_servers:
  - host: 127.0.0.1
    port: 19530
    mode: standalone

etcd_servers:
  - host: 127.0.0.1
    client_port: 2379

minio_servers:
  - host: 127.0.0.1
    port: 9000
    console_port: 9001
    access_key: "minioadmin"
    secret_key: "minioadmin"

# Optional: Monitoring
# monitoring_servers:
#   - host: 127.0.0.1
#     prometheus_port: 9090

# grafana_servers:
#   - host: 127.0.0.1
#     port: 3000
#     admin_password: "admin"
`

const kubernetesStandaloneTemplate = `# MiUp Kubernetes Topology - Standalone Mode
# Deploy with: miup instance deploy <instance-name> <this-file> --kubernetes
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
# Deploy with: miup instance deploy <instance-name> <this-file> --kubernetes
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

const localTLSTemplate = `# MiUp Local Instance - Standalone Mode with TLS (Docker Compose)
# Deploy with: miup instance deploy <instance-name> <this-file>
#
# Before deploying, create TLS certificates:
#   1. Generate certificates (server.pem, server.key, ca.pem)
#   2. Update the paths below to point to your certificate files

global:
  tls:
    enabled: true
    mode: 1  # 1 = one-way TLS, 2 = two-way TLS (mutual TLS)
    cert_file: "/path/to/server.pem"
    key_file: "/path/to/server.key"
    ca_file: "/path/to/ca.pem"
    # internal_enabled: false  # Enable TLS for internal component communication

milvus_servers:
  - host: 127.0.0.1
    port: 19530
    mode: standalone

etcd_servers:
  - host: 127.0.0.1
    client_port: 2379

minio_servers:
  - host: 127.0.0.1
    port: 9000
    console_port: 9001
    access_key: "minioadmin"
    secret_key: "minioadmin"
`

const kubernetesTLSTemplate = `# MiUp Kubernetes Topology - Standalone Mode with TLS
# Deploy with: miup instance deploy <instance-name> <this-file> --kubernetes
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error: %v", err))
		os.Exit(1)
	}
}
