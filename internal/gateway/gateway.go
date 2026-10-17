package gateway

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

type Manager struct {
	gatewayDir string
	port       int
	cmd        *exec.Cmd
	pidFile    string
}

func NewManager(gatewayDir string, port int) *Manager {
	homeDir, _ := os.UserHomeDir()
	pidFile := filepath.Join(homeDir, ".kirotoggle", "gateway.pid")

	return &Manager{
		gatewayDir: gatewayDir,
		port:       port,
		pidFile:    pidFile,
	}
}

func (m *Manager) Setup(apiKey, cliDBPath string) error {
	// Check if .venv exists, create if not
	venvPath := filepath.Join(m.gatewayDir, ".venv")
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		cmd := exec.Command("python3", "-m", "venv", ".venv")
		cmd.Dir = m.gatewayDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create venv: %w, output: %s", err, output)
		}
	}

	// Install requirements if not already installed
	// Check if site-packages exists and has content
	sitePackages := filepath.Join(venvPath, "lib")
	needsInstall := true
	if info, err := os.Stat(sitePackages); err == nil && info.IsDir() {
		// Check if there are any packages (more than just base pip/setuptools)
		entries, _ := os.ReadDir(sitePackages)
		if len(entries) > 0 {
			// For simplicity, we'll just check if site-packages exists with content
			// In a real implementation, we might parse requirements and check specifically
			needsInstall = false
		}
	}

	if needsInstall {
		pipPath := filepath.Join(venvPath, "bin", "pip")
		requirementsPath := filepath.Join(m.gatewayDir, "requirements.txt")
		cmd := exec.Command(pipPath, "install", "-r", requirementsPath)
		cmd.Dir = m.gatewayDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install requirements: %w, output: %s", err, output)
		}
	}

	// Write .env file
	envContent := fmt.Sprintf("PROXY_API_KEY=%s\nACCOUNT_SYSTEM=true\nKIRO_CLI_DB_FILE=%s\n", apiKey, cliDBPath)
	envPath := filepath.Join(m.gatewayDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

func (m *Manager) Start() error {
	// Ensure PID directory exists
	pidDir := filepath.Dir(m.pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Start the Python process
	pythonPath := filepath.Join(m.gatewayDir, ".venv", "bin", "python")
	mainPath := filepath.Join(m.gatewayDir, "main.py")

	m.cmd = exec.Command(pythonPath, mainPath)
	m.cmd.Dir = m.gatewayDir

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gateway process: %w", err)
	}

	// Store PID immediately
	if err := os.WriteFile(m.pidFile, []byte(strconv.Itoa(m.cmd.Process.Pid)), 0644); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Wait for TCP port to be listening (10s timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", m.port)
	tcpTimeout := time.After(10 * time.Second)
	tcpTicker := time.NewTicker(100 * time.Millisecond)
	defer tcpTicker.Stop()

	tcpReady := false
	for !tcpReady {
		select {
		case <-tcpTimeout:
			m.cmd.Process.Kill()
			os.Remove(m.pidFile)
			return fmt.Errorf("timeout waiting for TCP port %d to be ready", m.port)
		case <-tcpTicker.C:
			conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				tcpReady = true
			}
		}
	}

	// Wait for health endpoint to return healthy (20s timeout)
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", m.port)
	healthTimeout := time.After(20 * time.Second)
	healthTicker := time.NewTicker(200 * time.Millisecond)
	defer healthTicker.Stop()

	client := &http.Client{Timeout: 1 * time.Second}

	for {
		select {
		case <-healthTimeout:
			m.cmd.Process.Kill()
			os.Remove(m.pidFile)
			return fmt.Errorf("timeout waiting for health endpoint to be ready")
		case <-healthTicker.C:
			resp, err := client.Get(healthURL)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				var health map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&health); err == nil {
					if health["status"] == "healthy" {
						return nil
					}
				}
			}
		}
	}
}

func (m *Manager) IsRunning() bool {
	// Check if PID file exists
	pidData, err := os.ReadFile(m.pidFile)
	if err != nil {
		return false
	}

	// Parse PID
	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process is alive (doesn't actually send a signal)
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (m *Manager) Stop() error {
	// Read PID from file
	pidData, err := os.ReadFile(m.pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return fmt.Errorf("invalid PID in file: %w", err)
	}

	// Find process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		os.Remove(m.pidFile)
		return nil
	}

	// Wait up to 5 seconds for process to exit
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Force kill
			process.Signal(syscall.SIGKILL)
			os.Remove(m.pidFile)
			return nil
		case <-ticker.C:
			// Check if process is still alive
			err := process.Signal(syscall.Signal(0))
			if err != nil {
				// Process is dead
				os.Remove(m.pidFile)
				// Verify port is no longer listening
				addr := fmt.Sprintf("127.0.0.1:%d", m.port)
				conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
				if err == nil {
					conn.Close()
					return fmt.Errorf("port %d still listening after process stopped", m.port)
				}
				return nil
			}
		}
	}
}
