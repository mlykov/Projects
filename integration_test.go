//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain_Integration_NoFlags runs the compiled binary as a subprocess (no -lvm).
// Requires building the main binary; run in a privileged pod/container for full behavior.
func TestMain_Integration_NoFlags(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "test_binary")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testCmd := exec.CommandContext(ctx, binaryPath)
	var stdout, stderr bytes.Buffer
	testCmd.Stdout = &stdout
	testCmd.Stderr = &stderr

	err := testCmd.Run()

	if err != nil && ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("Unexpected error: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "=== Machine Info ===") {
		t.Errorf("Expected output to contain '=== Mashine Info ===', got:\n%s", output)
	}
	if !strings.Contains(output, "CPU cores:") {
		t.Errorf("Expected output to contain 'CPU cores:', got:\n%s", output)
	}
	if !strings.Contains(output, "Distribution:") {
		t.Errorf("Expected output to contain 'Distribution:', got:\n%s", output)
	}
	if strings.Contains(output, "=== LVM Procedure ===") {
		t.Error("Expected output to NOT contain LVM procedure when -lvm flag is not set")
	}
}

// TestMain_Integration_WithLVMFlag runs the compiled binary with -lvm.
// Run in a privileged pod/container for LVM and loop device access.
func TestMain_Integration_WithLVMFlag(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "test_binary")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testCmd := exec.CommandContext(ctx, binaryPath, "-lvm")
	var stdout, stderr bytes.Buffer
	testCmd.Stdout = &stdout
	testCmd.Stderr = &stderr

	err := testCmd.Run()

	if err != nil && ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("Unexpected error: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "=== Machine Info ===") {
		t.Errorf("Expected output to contain '=== Mashine Info ===', got:\n%s", output)
	}
	if !strings.Contains(output, "=== Running LVM Procedure ===") {
		t.Error("Expected output to contain '=== Running LVM Procedure ===' when -lvm flag is set")
	}
}
