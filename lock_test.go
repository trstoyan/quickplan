package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestLockAcquisition(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quickplan-lock-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-project"
	projectPath := filepath.Join(tmpDir, projectName)
	err = os.MkdirAll(projectPath, 0755)
	if err != nil {
		t.Fatal(err)
	}

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	// 1. Acquire lock
	err = pdm.AcquireLock(projectName, 10)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// 2. Check if file exists
	lockPath := pdm.getLockPath(projectName)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("Lock file does not exist")
	}

	// 3. Try to acquire again (should fail)
	err = pdm.AcquireLock(projectName, 10)
	if err == nil {
		t.Fatal("Expected failure when acquiring held lock, but got success")
	}

	// 4. Release lock
	err = pdm.ReleaseLock(projectName)
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// 5. Check if file is gone
	if _, err := os.Stat(lockPath); err == nil {
		t.Fatal("Lock file still exists after release")
	}
}

func TestStaleLockExpiry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quickplan-lock-stale-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "stale-project"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	// Manually create an expired lock
	host, _ := os.Hostname()
	lock := Lock{
		PID:       12345,
		Host:      host,
		CreatedAt: time.Now().Add(-1 * time.Hour),
		TTL:       300,
	}
	data, _ := yaml.Marshal(lock)
	os.WriteFile(pdm.getLockPath(projectName), data, 0644)

	// Try to acquire (should succeed by overriding stale lock)
	err = pdm.AcquireLock(projectName, 10)
	if err != nil {
		t.Fatalf("Failed to acquire stale lock: %v", err)
	}

	stale, currentLock, _ := pdm.IsLockStale(projectName)
	if stale {
		t.Error("Lock should not be stale after fresh acquisition")
	}
	if currentLock.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), currentLock.PID)
	}
}

func TestLockHostMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quickplan-lock-host-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "host-mismatch"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	// Manually create a lock from a different host
	lock := Lock{
		PID:       os.Getpid(), // Current PID but different host
		Host:      "other-host",
		CreatedAt: time.Now(),
		TTL:       3600,
	}
	data, _ := yaml.Marshal(lock)
	os.WriteFile(pdm.getLockPath(projectName), data, 0644)

	// Try to acquire (should fail because it's not stale by time, and host differs so we can't verify PID)
	err = pdm.AcquireLock(projectName, 10)
	if err == nil {
		t.Fatal("Expected failure when acquiring lock from other host, but got success")
	}
}
