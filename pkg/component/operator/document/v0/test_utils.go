package document

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	// Cache for test file contents to avoid repeated I/O
	testFileCache = make(map[string][]byte)
	testFileMutex sync.RWMutex
)

// checkExternalDependency checks if an external command is available
func checkExternalDependency(cmd string) bool {
	_, err := exec.LookPath(cmd)
	if err != nil {
		return false
	}

	// For LibreOffice, also check if it can run without errors
	if cmd == "libreoffice" {
		return checkLibreOfficeHealth()
	}

	// For Python, also check if the virtual environment Python exists
	if cmd == "python3" || cmd == "python" {
		_, err := exec.LookPath("/opt/venv/bin/python")
		return err == nil
	}

	return true
}

// checkLibreOfficeHealth performs a basic health check for LibreOffice
func checkLibreOfficeHealth() bool {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "libreoffice_health_check")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tempDir)

	// Set proper permissions
	if err := os.Chmod(tempDir, 0755); err != nil {
		return false
	}

	// Try to run LibreOffice with --version to check if it's working
	cmd := exec.Command("libreoffice", "--headless", "--version")
	cmd.Env = append(os.Environ(), "HOME="+tempDir)

	// Set a timeout to avoid hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, "libreoffice", "--headless", "--version")
	cmd.Env = append(os.Environ(), "HOME="+tempDir)

	err = cmd.Run()
	return err == nil
}

// getTestFileContent returns cached file content or reads it once
func getTestFileContent(filepath string) ([]byte, error) {
	testFileMutex.RLock()
	if content, exists := testFileCache[filepath]; exists {
		testFileMutex.RUnlock()
		return content, nil
	}
	testFileMutex.RUnlock()

	testFileMutex.Lock()
	defer testFileMutex.Unlock()

	// Double-check after acquiring write lock
	if content, exists := testFileCache[filepath]; exists {
		return content, nil
	}

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	testFileCache[filepath] = content
	return content, nil
}

// clearTestFileCache clears the test file cache (useful for testing)
func clearTestFileCache() {
	testFileMutex.Lock()
	defer testFileMutex.Unlock()
	testFileCache = make(map[string][]byte)
}
