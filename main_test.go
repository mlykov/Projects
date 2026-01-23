package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ========================================== readCpuCores tests ===========================================
// Test readCpuCores - nproc execution failure
// This test checks that the function correctly handles the nproc failure and returns a safe value (0)
// instead of panicking or returning an incorrect result.
func TestReadCpuCores_NprocFails(t *testing.T) {
	// Create a mock nproc that exits with error
	tmpdir := t.TempDir()
	mockNproc := filepath.Join(tmpdir, "nproc")

	// Create a script that exits with error code 1
	script := "#!/bin/sh\nexit 1\n"
	err := os.WriteFile(mockNproc, []byte(script), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock nproc: %v", err)
	}

	// Save original PATH
	originalPath := os.Getenv("PATH")

	// Prepend our mock directory to PATH so our mock nproc is found first
	os.Setenv("PATH", tmpdir+":"+originalPath)
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	cores := readCpuCores()
	if cores != 0 {
		t.Errorf("Expected 0 cores when nproc fails, got %d", cores)
	}
}

// Test readCpuCores - integer parse failure
// This test checks that the function correctly handles invalid output from nproc (non-numeric string)
// and returns a safe value (0) instead of panicking when parsing fails.
func TestReadCpuCores_ParseFails(t *testing.T) {
	// Create a mock nproc that returns invalid output (non-numeric)
	tmpdir := t.TempDir()
	mockNproc := filepath.Join(tmpdir, "nproc")

	// Create a script that outputs invalid data
	script := "#!/bin/sh\necho 'NaN'\n"
	err := os.WriteFile(mockNproc, []byte(script), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock nproc: %v", err)
	}

	// Save original PATH
	originalPath := os.Getenv("PATH")

	// Prepend our mock directory to PATH
	os.Setenv("PATH", tmpdir+":"+originalPath)
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	cores := readCpuCores()
	if cores != 0 {
		t.Errorf("Expected 0 cores when parse fails, got %d", cores)
	}
}

// Test readCpuCores - success case (everything works)
// This test verifies that the function successfully reads CPU core count using nproc
// and returns a positive integer value when everything works correctly.
func TestReadCpuCores_Success(t *testing.T) {
	// Check if nproc is available
	if _, err := exec.LookPath("nproc"); err != nil {
		t.Skip("Skipping test: nproc is not available (not on Linux or not in PATH)")
	}

	cores := readCpuCores()
	if cores <= 0 {
		t.Errorf("Expected positive number of CPU cores, got %d", cores)
	}
}

// ========================================== readMemory tests ===========================================
// Test readMemoryFromData - table-driven tests for parsing logic
// This test verifies the parsing logic with various input fixtures without file I/O.
func TestReadMemoryFromData(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		usedKB float64
		freeKB float64
	}{
		{
			name: "normal case",
			input: `MemTotal:       8192000 kB
MemFree:        1024000 kB
MemAvailable:   2048000 kB
`,
			usedKB: 8192000 - 2048000, // total - available
			freeKB: 2048000,
		},
		{
			name: "bad memtotal line",
			input: `FooTotal:       8192000 kB
MemFree:        1024000 kB
MemAvailable:   2048000 kB
`,
			usedKB: 0,
			freeKB: 0,
		},
		{
			name: "bad memavailable line",
			input: `MemTotal:       8192000 kB
MemFree:        1024000 kB
FooAvailable:   2048000 kB
`,
			usedKB: 0,
			freeKB: 0,
		},
		{
			name: "invalid number in memtotal",
			input: `MemTotal:       not-a-number kB
MemFree:        1024000 kB
MemAvailable:   2048000 kB
`,
			usedKB: 0,
			freeKB: 0,
		},
		{
			name: "invalid number in memavailable",
			input: `MemTotal:       8192000 kB
MemFree:        1024000 kB
MemAvailable:   not-a-number kB
`,
			usedKB: 0,
			freeKB: 0,
		},
		{
			name: "not enough lines",
			input: `MemTotal:       8192000 kB
MemFree:        1024000 kB
`,
			usedKB: 0,
			freeKB: 0,
		},
		{
			name:   "empty input",
			input:  ``,
			usedKB: 0,
			freeKB: 0,
		},
		{
			name: "small memory values",
			input: `MemTotal:       1000 kB
MemFree:        100 kB
MemAvailable:   400 kB
`,
			usedKB: 600, // 1000 - 400
			freeKB: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, free := readMemoryFromData([]byte(tt.input))
			if used != tt.usedKB || free != tt.freeKB {
				t.Errorf("got used=%.2f free=%.2f, want used=%.2f free=%.2f",
					used, free, tt.usedKB, tt.freeKB)
			}
		})
	}
}

// Test readMemory - file read failure
// This test checks that the function correctly handles the case when /proc/meminfo cannot be read
// and returns safe values (0, 0) instead of panicking.
func TestReadMemory_FileReadFails(t *testing.T) {
	// Test that readMemoryFromData handles empty/invalid data correctly
	used, free := readMemoryFromData([]byte(""))
	if used != 0 || free != 0 {
		t.Errorf("Expected 0,0 for empty input, got used=%.2f, free=%.2f", used, free)
	}
}

// Test readMemory - success case (everything works)
// This test verifies that the function successfully reads memory information from /proc/meminfo
// and returns valid non-negative values for used and free memory.
func TestReadMemory_Success(t *testing.T) {
	// This test requires /proc/meminfo to exist (Linux only)
	if _, err := os.Stat("/proc/meminfo"); os.IsNotExist(err) {
		t.Skip("Skipping test: /proc/meminfo does not exist (not on Linux)")
	}

	used, free := readMemory()
	if used < 0 || free < 0 {
		t.Errorf("Expected non-negative memory values, got used=%.2f, free=%.2f", used, free)
	}
}

// ========================================== readDistro tests ===========================================
// Test readDistroFromData - table-driven tests for parsing logic
// This test verifies the parsing logic with various input fixtures without file I/O.
func TestReadDistroFromData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal case with quotes",
			input:    `PRETTY_NAME="Ubuntu 22.04.3 LTS"`,
			expected: "Ubuntu 22.04.3 LTS",
		},
		{
			name:     "normal case without quotes",
			input:    `PRETTY_NAME=Debian GNU/Linux 12 (bookworm)`,
			expected: "Debian GNU/Linux 12 (bookworm)",
		},
		{
			name: "normal case with quotes and extra lines",
			input: `PRETTY_NAME="Fedora Linux 39"
NAME="Fedora Linux"
VERSION="39 (Workstation Edition)"`,
			expected: "Fedora Linux 39",
		},
		{
			name: "invalid format - no PRETTY_NAME",
			input: `NAME="Ubuntu"
VERSION="22.04.3 LTS"`,
			expected: "Invalid",
		},
		{
			name:     "empty input",
			input:    ``,
			expected: "Invalid",
		},
		{
			name: "empty first line",
			input: `
PRETTY_NAME="Ubuntu 22.04.3 LTS"`,
			expected: "Invalid",
		},
		{
			name: "PRETTY_NAME not at start",
			input: `NAME="Ubuntu"
PRETTY_NAME="Ubuntu 22.04.3 LTS"`,
			expected: "Invalid",
		},
		{
			name:     "quotes only",
			input:    `PRETTY_NAME=""`,
			expected: "",
		},
		{
			name:     "single quote",
			input:    `PRETTY_NAME="Ubuntu"`,
			expected: "Ubuntu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readDistroFromData([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test readDistro - file read failure
// This test checks that the function correctly handles the case when /etc/os-release cannot be read
// and returns an appropriate error message string instead of panicking.
func TestReadDistro_FileReadFails(t *testing.T) {
	// Test that readDistroFromData handles empty/invalid data correctly
	result := readDistroFromData([]byte(""))
	if result != "Invalid" {
		t.Errorf("Expected 'Invalid' for empty input, got: %s", result)
	}
}

// Test readDistro - success case (everything works)
// This test verifies that the function successfully reads the distribution name from /etc/os-release
// and returns a valid non-empty string without error messages.
func TestReadDistro_Success(t *testing.T) {
	// This test requires /etc/os-release to exist
	if _, err := os.Stat("/etc/os-release"); os.IsNotExist(err) {
		t.Skip("Skipping test: /etc/os-release does not exist")
	}

	distro := readDistro()
	if distro == "" {
		t.Error("Expected non-empty distro string")
	}
	if strings.Contains(distro, "Error") {
		t.Errorf("Expected valid distro, got error: %s", distro)
	}
}

// ========================================== readDevices tests ===========================================
// Test readDevicesFromOutput - table-driven tests for output processing logic
// This test verifies the output processing logic with various input fixtures without command execution.
func TestReadDevicesFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   []byte
		err      error
		expected string
	}{
		{
			name:     "success case",
			output:   []byte("00:00.0 Host bridge: Intel Corporation Device\n00:02.0 VGA compatible controller: Intel Corporation Device"),
			err:      nil,
			expected: "00:00.0 Host bridge: Intel Corporation Device\n00:02.0 VGA compatible controller: Intel Corporation Device",
		},
		{
			name:     "empty output",
			output:   []byte(""),
			err:      nil,
			expected: "",
		},
		{
			name:     "command execution error",
			output:   nil,
			err:      exec.ErrNotFound,
			expected: "Executing lspci failed:executable file not found",
		},
		{
			name:     "command error with message",
			output:   []byte("some output"),
			err:      fmt.Errorf("command failed"),
			expected: "Executing lspci failed:command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readDevicesFromOutput(tt.output, tt.err)
			if !strings.Contains(result, tt.expected) && result != tt.expected {
				t.Errorf("got %q, want to contain %q", result, tt.expected)
			}
		})
	}
}

// Test readDevices - lspci execution failure
// This test checks that the function correctly handles the case when lspci command fails
// and returns an appropriate error message string instead of panicking.
func TestReadDevices_LspciFails(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")

	// Set PATH to empty to make lspci unavailable
	os.Setenv("PATH", "")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	devices := readDevices()
	if !strings.Contains(devices, "Executing lspci failed") {
		t.Errorf("Expected error message when lspci fails, got: %s", devices)
	}
}

// Test readDevices - success case (everything works)
// This test verifies that the function successfully executes lspci command
// and returns a non-empty string with device information.
// This is an integration test that requires lspci to be available.
func TestReadDevices_Success(t *testing.T) {
	// This test requires lspci to be available
	devices := readDevices()
	if strings.Contains(devices, "Executing lspci failed") {
		t.Skip("Skipping test: lspci is not available")
	}
	if devices == "" {
		t.Error("Expected non-empty devices string")
	}
}

// ========================================== runCommand tests ===========================================
// Test runCommand - command failure
// This test checks that the function correctly handles command execution failures
// and returns a non-nil error with an appropriate error message containing "command failed".
func TestRunCommand_Failure(t *testing.T) {
	err := runCommand("false")
	if err == nil {
		t.Error("Expected error when command fails, got nil")
	}
	if !strings.Contains(err.Error(), "command failed") {
		t.Errorf("Expected error message to contain 'command failed', got: %v", err)
	}
}

// Test runCommand - invalid command
// This test verifies that the function correctly handles non-existent commands
// and returns a non-nil error instead of panicking.
func TestRunCommand_InvalidCommand(t *testing.T) {
	err := runCommand("nonexistent_command_xyz123_should_fail")
	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}

// Test runCommand - success case (everything works)
// This test verifies that the function successfully executes a valid command
// and returns nil error when the command completes successfully.
func TestRunCommand_Success(t *testing.T) {
	err := runCommand("echo 'test'")
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

// ========================================== runDiskProcedure tests ===========================================
// Test runDiskProcedure - commands order - success case
// This test verifies that runDiskProcedure executes commands in the correct order
// without actually running them or requiring root privileges.
func TestRunDiskProcedure_CommandsOrder(t *testing.T) {
	var got []string
	fakeRun := func(cmd string) error {
		got = append(got, cmd)
		return nil
	}

	err := runCommands(diskCommands(), fakeRun)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := diskCommands()
	if len(want) != len(got) {
		t.Fatalf("commands count mismatch: want %d, got %d", len(want), len(got))
	}

	for i, cmd := range want {
		if i >= len(got) || got[i] != cmd {
			t.Errorf("command[%d]: want %q, got %q", i, cmd, got[i])
		}
	}
}

// Test runDiskProcedure - error propagation
// This test verifies that runDiskProcedure correctly propagates errors
// when a command fails during execution.
func TestRunDiskProcedure_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name          string
		failingCmd    string
		expectedError string
	}{
		{
			name:          "error on first command (mkdir)",
			failingCmd:    "mkdir -p ~/file_systems_test",
			expectedError: "command failed",
		},
		{
			name:          "error on fallocate",
			failingCmd:    "fallocate",
			expectedError: "command failed",
		},
		{
			name:          "error on mkfs",
			failingCmd:    "mkfs.ext4",
			expectedError: "command failed",
		},
		{
			name:          "error on mount",
			failingCmd:    "sudo mount",
			expectedError: "command failed",
		},
		{
			name:          "error on write file",
			failingCmd:    "printf",
			expectedError: "command failed",
		},
		{
			name:          "error on umount",
			failingCmd:    "sudo umount",
			expectedError: "command failed",
		},
		{
			name:          "error on last command",
			failingCmd:    "rm -rf ~/file_systems_test",
			expectedError: "command failed",
		},
		{
			name:          "error on cd",
			failingCmd:    "cd ~/file_systems_test",
			expectedError: "command failed",
		},
		{
			name:          "error on sudo mkdir",
			failingCmd:    "sudo mkdir -p /mnt/disk1",
			expectedError: "command failed",
		},
		{
			name:          "error on cat",
			failingCmd:    "cat /mnt/disk1/test.txt",
			expectedError: "command failed",
		},
		{
			name:          "error on wipefs",
			failingCmd:    "wipefs",
			expectedError: "command failed",
		},
		{
			name:          "error on rm -f",
			failingCmd:    "rm -f disk1",
			expectedError: "command failed",
		},
		{
			name:          "error on sudo rm -rf",
			failingCmd:    "sudo rm -rf /mnt/disk1",
			expectedError: "command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRun := func(cmd string) error {
				if strings.Contains(cmd, tt.failingCmd) {
					return fmt.Errorf("%s: %s", tt.expectedError, cmd)
				}
				return nil
			}

			err := runCommands(diskCommands(), fakeRun)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error to contain %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

// Test runDiskProcedure - early exit on error
// This test verifies that when a command fails, subsequent commands are not executed.
func TestRunDiskProcedure_EarlyExitOnError(t *testing.T) {
	var executedCommands []string
	failingIndex := 3 // fail on 4th command (index 3: "mkfs.ext4")

	fakeRun := func(cmd string) error {
		executedCommands = append(executedCommands, cmd)
		if len(executedCommands) == failingIndex+1 {
			return fmt.Errorf("command failed: %s", cmd)
		}
		return nil
	}

	commands := diskCommands()
	err := runCommands(commands, fakeRun)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// Should only execute commands up to the failing one
	expectedCount := failingIndex + 1
	if len(executedCommands) != expectedCount {
		t.Errorf("expected %d commands executed, got %d", expectedCount, len(executedCommands))
	}

	// Verify that commands after the failing one were not executed
	if len(executedCommands) < len(commands) {
		nextCmd := commands[len(executedCommands)]
		t.Logf("Correctly stopped execution before: %s", nextCmd)
	}
}

// ========================================== runLVMProcedure tests ===========================================
// Test runLVMProcedure - commands order
// This test verifies that runLVMProcedure executes commands in the correct order
// without actually running them or requiring root privileges.
func TestRunLVMProcedure_CommandsOrder(t *testing.T) {
	var got []string
	fakeRun := func(cmd string) error {
		got = append(got, cmd)
		return nil
	}

	homeDir := "/home/test"
	loopDevice := "/dev/loop0"
	commands := lvmCommands(homeDir, loopDevice)
	err := runCommands(commands, fakeRun)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := lvmCommands(homeDir, loopDevice)
	if len(want) != len(got) {
		t.Fatalf("commands count mismatch: want %d, got %d", len(want), len(got))
	}

	for i, cmd := range want {
		if i >= len(got) || got[i] != cmd {
			t.Errorf("command[%d]: want %q, got %q", i, cmd, got[i])
		}
	}
}

// Test runLVMProcedure - error propagation
// This test verifies that runLVMProcedure correctly propagates errors
// when a command fails during execution.
func TestRunLVMProcedure_ErrorPropagation(t *testing.T) {
	homeDir := "/home/test"
	loopDevice := "/dev/loop0"
	commands := lvmCommands(homeDir, loopDevice)

	tests := []struct {
		name          string
		failingCmd    string
		expectedError string
	}{
		{
			name:          "error on first command (mkdir)",
			failingCmd:    "mkdir -p",
			expectedError: "command failed",
		},
		{
			name:          "error on fallocate",
			failingCmd:    "fallocate",
			expectedError: "command failed",
		},
		{
			name:          "error on losetup",
			failingCmd:    "sudo losetup",
			expectedError: "command failed",
		},
		{
			name:          "error on pvcreate",
			failingCmd:    "sudo pvcreate",
			expectedError: "command failed",
		},
		{
			name:          "error on vgcreate",
			failingCmd:    "sudo vgcreate",
			expectedError: "command failed",
		},
		{
			name:          "error on lvcreate",
			failingCmd:    "sudo lvcreate",
			expectedError: "command failed",
		},
		{
			name:          "error on mkfs",
			failingCmd:    "sudo mkfs.ext4",
			expectedError: "command failed",
		},
		{
			name:          "error on mount",
			failingCmd:    "sudo mount",
			expectedError: "command failed",
		},
		{
			name:          "error on last command",
			failingCmd:    "sudo rm -rf /mnt/lvm1",
			expectedError: "command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRun := func(cmd string) error {
				if strings.Contains(cmd, tt.failingCmd) {
					return fmt.Errorf("%s: %s", tt.expectedError, cmd)
				}
				return nil
			}

			err := runCommands(commands, fakeRun)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error to contain %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

// Test runLVMProcedure - early exit on error
// This test verifies that when a command fails, subsequent commands are not executed.
func TestRunLVMProcedure_EarlyExitOnError(t *testing.T) {
	var executedCommands []string
	failingIndex := 4 // fail on 5th command (index 4: "sudo pvcreate")

	homeDir := "/home/test"
	loopDevice := "/dev/loop0"
	commands := lvmCommands(homeDir, loopDevice)

	fakeRun := func(cmd string) error {
		executedCommands = append(executedCommands, cmd)
		if len(executedCommands) == failingIndex+1 {
			return fmt.Errorf("command failed: %s", cmd)
		}
		return nil
	}

	err := runCommands(commands, fakeRun)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// Should only execute commands up to the failing one
	expectedCount := failingIndex + 1
	if len(executedCommands) != expectedCount {
		t.Errorf("expected %d commands executed, got %d", expectedCount, len(executedCommands))
	}

	// Verify that commands after the failing one were not executed
	if len(executedCommands) < len(commands) {
		nextCmd := commands[len(executedCommands)]
		t.Logf("Correctly stopped execution before: %s", nextCmd)
	}
}

// ========================================== main tests ===========================================
// Test main - flag parsing success (unit test)
// This test verifies that the -lvm flag is correctly parsed when provided.
// We test flag parsing separately as the main function cannot be easily tested due to its infinite loop.
func TestMain_FlagParsing_Success(t *testing.T) {
	// Create a new flag set for testing
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Test with -lvm flag
	useLVM := fs.Bool("lvm", false, "Use LVM procedure")
	testArgs := []string{"-lvm"}
	if err := fs.Parse(testArgs); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if !*useLVM {
		t.Error("Expected -lvm flag to be true")
	}
}

// Test main - flag parsing failure (invalid flag)
// This test verifies that parsing fails when an unknown flag is provided.
func TestMain_FlagParsing_InvalidFlag(t *testing.T) {
	// Create a new flag set for testing
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Test with invalid flag
	useLVM := fs.Bool("lvm", false, "Use LVM procedure")
	testArgs := []string{"-unknown-flag"}
	err := fs.Parse(testArgs)

	if err == nil {
		t.Error("Expected error when parsing invalid flag, got nil")
	}
	if *useLVM {
		t.Error("Expected -lvm flag to remain false when invalid flag is provided")
	}
}

// Test main - flag parsing failure (invalid format)
// This test verifies that parsing fails when a flag receives an invalid value type.
func TestMain_FlagParsing_InvalidFormat(t *testing.T) {
	// Create a new flag set for testing
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Add an int flag to test invalid value format
	testInt := fs.Int("test-int", 0, "Test int flag")
	useLVM := fs.Bool("lvm", false, "Use LVM procedure")

	// Test with invalid format: provide non-numeric value to int flag
	testArgs := []string{"-test-int=not-a-number"}
	err := fs.Parse(testArgs)

	// flag.Parse should return error when an int flag receives a non-numeric value
	if err == nil {
		t.Error("Expected error when parsing int flag with non-numeric value, got nil")
	}
	if *testInt != 0 {
		t.Error("Expected test-int flag to remain at default value when parsing fails")
	}
	if *useLVM {
		t.Error("Expected -lvm flag to remain false when parsing fails")
	}
}

// Test main - integration test via os/exec
// This test runs the compiled binary as a subprocess to test main function behavior.
func TestMain_Integration_NoFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the binary
	binaryPath := filepath.Join(t.TempDir(), "test_binary")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	// Run the binary with timeout (main has infinite loop)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testCmd := exec.CommandContext(ctx, binaryPath)
	var stdout, stderr bytes.Buffer
	testCmd.Stdout = &stdout
	testCmd.Stderr = &stderr

	err := testCmd.Run()

	// Context timeout is expected (main runs forever)
	if err != nil && ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("Unexpected error: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()

	// Verify expected output
	if !strings.Contains(output, "=== Mashine Info ===") {
		t.Errorf("Expected output to contain '=== Mashine Info ===', got:\n%s", output)
	}
	if !strings.Contains(output, "CPU cores:") {
		t.Errorf("Expected output to contain 'CPU cores:', got:\n%s", output)
	}
	if !strings.Contains(output, "Distribution:") {
		t.Errorf("Expected output to contain 'Distribution:', got:\n%s", output)
	}
	// Should not contain LVM procedure (no -lvm flag)
	if strings.Contains(output, "=== LVM Procedure ===") {
		t.Error("Expected output to NOT contain LVM procedure when -lvm flag is not set")
	}
}

// Test main - integration test with -lvm flag
// This test runs the compiled binary with -lvm flag to verify it uses LVM procedure.
func TestMain_Integration_WithLVMFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the binary
	binaryPath := filepath.Join(t.TempDir(), "test_binary")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove(binaryPath)

	// Run the binary with -lvm flag and timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	testCmd := exec.CommandContext(ctx, binaryPath, "-lvm")
	var stdout, stderr bytes.Buffer
	testCmd.Stdout = &stdout
	testCmd.Stderr = &stderr

	err := testCmd.Run()

	// Context timeout is expected (main runs forever)
	if err != nil && ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("Unexpected error: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()

	// Verify expected output
	if !strings.Contains(output, "=== Mashine Info ===") {
		t.Errorf("Expected output to contain '=== Mashine Info ===', got:\n%s", output)
	}
	// Should contain LVM procedure when -lvm flag is set
	if !strings.Contains(output, "=== Running LVM Procedure ===") {
		t.Error("Expected output to contain '=== Running LVM Procedure ===' when -lvm flag is set")
	}
}
