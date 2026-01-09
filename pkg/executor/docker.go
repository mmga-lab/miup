package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DockerCompose provides docker-compose operations
type DockerCompose struct {
	workDir     string
	projectName string
}

// NewDockerCompose creates a new DockerCompose instance
func NewDockerCompose(workDir, projectName string) *DockerCompose {
	return &DockerCompose{
		workDir:     workDir,
		projectName: projectName,
	}
}

// WorkDir returns the working directory
func (dc *DockerCompose) WorkDir() string {
	return dc.workDir
}

// ProjectName returns the project name
func (dc *DockerCompose) ProjectName() string {
	return dc.projectName
}

// ComposeFilePath returns the path to the docker-compose.yaml file
func (dc *DockerCompose) ComposeFilePath() string {
	return filepath.Join(dc.workDir, "docker-compose.yaml")
}

// Up starts the compose services
func (dc *DockerCompose) Up(ctx context.Context) error {
	return dc.run(ctx, "up", "-d", "--remove-orphans", "--wait")
}

// Down stops and removes the compose services
func (dc *DockerCompose) Down(ctx context.Context, removeVolumes bool) error {
	args := []string{"down"}
	if removeVolumes {
		args = append(args, "-v")
	}
	return dc.run(ctx, args...)
}

// Stop stops the compose services without removing them
func (dc *DockerCompose) Stop(ctx context.Context) error {
	return dc.run(ctx, "stop")
}

// Start starts existing compose services
func (dc *DockerCompose) Start(ctx context.Context) error {
	return dc.run(ctx, "start")
}

// PS lists compose services and returns the output
func (dc *DockerCompose) PS(ctx context.Context) (string, error) {
	return dc.runOutput(ctx, "ps", "--format", "table")
}

// Logs gets compose service logs
func (dc *DockerCompose) Logs(ctx context.Context, service string, tail int) (string, error) {
	args := []string{"logs", "--tail", fmt.Sprintf("%d", tail)}
	if service != "" {
		args = append(args, service)
	}
	return dc.runOutput(ctx, args...)
}

// IsRunning checks if compose services are running
func (dc *DockerCompose) IsRunning(ctx context.Context) (bool, error) {
	output, err := dc.runOutput(ctx, "ps", "-q")
	if err != nil {
		// If the project doesn't exist, it's not running
		return false, nil
	}
	return strings.TrimSpace(output) != "", nil
}

// WaitForHealthy waits for all services to be healthy
func (dc *DockerCompose) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		output, err := dc.runOutput(ctx, "ps", "--format", "json")
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				continue
			}
		}

		// Simple check: if we have output and no "starting" or "unhealthy"
		if output != "" && !strings.Contains(strings.ToLower(output), "starting") &&
			!strings.Contains(strings.ToLower(output), "unhealthy") {
			// Additional check for running state
			if strings.Contains(strings.ToLower(output), "running") {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			continue
		}
	}

	return fmt.Errorf("timeout waiting for services to be healthy")
}

// Exists checks if the compose file exists
func (dc *DockerCompose) Exists() bool {
	_, err := os.Stat(dc.ComposeFilePath())
	return err == nil
}

// run executes a docker compose command
func (dc *DockerCompose) run(ctx context.Context, args ...string) error {
	cmd := dc.buildCommand(ctx, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runOutput executes a docker compose command and returns output
func (dc *DockerCompose) runOutput(ctx context.Context, args ...string) (string, error) {
	cmd := dc.buildCommand(ctx, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runSilent executes a docker compose command silently
func (dc *DockerCompose) runSilent(ctx context.Context, args ...string) error {
	cmd := dc.buildCommand(ctx, args...)
	return cmd.Run()
}

// buildCommand builds the docker compose command
func (dc *DockerCompose) buildCommand(ctx context.Context, args ...string) *exec.Cmd {
	baseArgs := []string{"compose", "-f", dc.ComposeFilePath(), "-p", dc.projectName}
	baseArgs = append(baseArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", baseArgs...)
	cmd.Dir = dc.workDir
	return cmd
}

// CheckDockerAvailable checks if docker is available
func CheckDockerAvailable() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}
	return nil
}

// CheckDockerComposeAvailable checks if docker compose is available
func CheckDockerComposeAvailable() error {
	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose is not available: %w", err)
	}
	return nil
}

// CheckDockerRunning checks if docker daemon is running
func CheckDockerRunning() error {
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon is not running: %w", err)
	}
	return nil
}
