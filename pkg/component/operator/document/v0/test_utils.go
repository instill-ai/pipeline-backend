package document

import (
	"os"
	"os/exec"
	"sync"
)

var (
	// Cache for test file contents to avoid repeated I/O
	testFileCache = make(map[string][]byte)
	testFileMutex sync.RWMutex
)

// checkExternalDependency checks if an external command is available
func checkExternalDependency(cmd string) bool {
	_, err := exec.LookPath(cmd)
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
