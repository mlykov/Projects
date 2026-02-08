package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
)

// ===================== readCpuCores =====================
func TestReadCpuCores(t *testing.T) {
	oldExec := ExecOutput
	defer func() { ExecOutput = oldExec }()

	tests := []struct {
		name      string
		mockOut   []byte
		mockErr   error
		wantCores int
		success   bool
	}{
		{
			name:      "success: valid nproc output",
			mockOut:   []byte("8"),
			mockErr:   nil,
			wantCores: 8,
			success:   true,
		},
		{
			name:      "failure: nproc execution fails",
			mockOut:   nil,
			mockErr:   errors.New("exec failed"),
			wantCores: 0,
			success:   false,
		},
		{
			name:      "failure: parse fails (non-numeric)",
			mockOut:   []byte("NaN"),
			mockErr:   nil,
			wantCores: 0,
			success:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ExecOutput = func(name string, _ ...string) ([]byte, error) {
				if name == "nproc" {
					return tt.mockOut, tt.mockErr
				}
				return nil, nil
			}
			got := readCpuCores()
			if got != tt.wantCores {
				t.Errorf("readCpuCores() = %d, want %d", got, tt.wantCores)
			}
		})
	}
	ExecOutput = oldExec
}

// ===================== readMemoryFromData (pure) =====================
func TestReadMemoryFromData(t *testing.T) {
	// Fake /proc/meminfo content: 8GB total, 2GB available.
	sampleMeminfo8GB := "MemTotal:       8192000 kB\nMemFree:        1024000 kB\nMemAvailable:   2048000 kB\n"
	// Fake /proc/meminfo with small numbers (1MB total, 400KB available).
	sampleMeminfoSmall := "MemTotal:       1000 kB\nMemFree:        100 kB\nMemAvailable:   400 kB\n"

	tests := []struct {
		name   string
		input  string
		usedKB float64
		freeKB float64
	}{
		{"success: normal case", sampleMeminfo8GB, 8192000 - 2048000, 2048000},
		{"failure: bad memtotal line", "FooTotal:       8192000 kB\nMemFree:        1024000 kB\nMemAvailable:   2048000 kB\n", 0, 0},
		{"failure: bad memavailable line", "MemTotal:       8192000 kB\nMemFree:        1024000 kB\nFooAvailable:   2048000 kB\n", 0, 0},
		{"failure: invalid number in memtotal", "MemTotal:       not-a-number kB\nMemFree:        1024000 kB\nMemAvailable:   2048000 kB\n", 0, 0},
		{"failure: invalid number in memavailable", "MemTotal:       8192000 kB\nMemFree:        1024000 kB\nMemAvailable:   not-a-number kB\n", 0, 0},
		{"failure: not enough lines", "MemTotal:       8192000 kB\nMemFree:        1024000 kB\n", 0, 0},
		{"failure: empty input", "", 0, 0},
		{"success: small memory values", sampleMeminfoSmall, 600, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, free := readMemoryFromData([]byte(tt.input))
			if used != tt.usedKB || free != tt.freeKB {
				t.Errorf("readMemoryFromData() = used %.2f free %.2f, want used %.2f free %.2f",
					used, free, tt.usedKB, tt.freeKB)
			}
		})
	}
}

// ===================== readMemory =====================
func TestReadMemory(t *testing.T) {
	oldReadFile := ReadFile
	defer func() { ReadFile = oldReadFile }()

	tests := []struct {
		name     string
		mockData []byte
		mockErr  error
		wantUsed float64
		wantFree float64
		success  bool
	}{
		{
			name:     "success: valid meminfo",
			mockData: []byte("MemTotal:       8192000 kB\nMemFree:        1024000 kB\nMemAvailable:   2048000 kB\n"),
			mockErr:  nil,
			wantUsed: 6144000,
			wantFree: 2048000,
			success:  true,
		},
		{
			name:     "failure: file read error",
			mockData: nil,
			mockErr:  errors.New("file not found"),
			wantUsed: 0,
			wantFree: 0,
			success:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ReadFile = func(path string) ([]byte, error) {
				if path != "/proc/meminfo" {
					return nil, errors.New("unexpected path")
				}
				return tt.mockData, tt.mockErr
			}
			used, free := readMemory()
			if used != tt.wantUsed || free != tt.wantFree {
				t.Errorf("readMemory() = used %.2f free %.2f, want used %.2f free %.2f",
					used, free, tt.wantUsed, tt.wantFree)
			}
		})
	}
	ReadFile = oldReadFile
}

// ===================== readDistroFromData =====================
func TestReadDistroFromData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		success  bool
	}{
		{
			name:     "success: normal case with quotes",
			input:    `PRETTY_NAME="Ubuntu 22.04.3 LTS"`,
			expected: "Ubuntu 22.04.3 LTS",
			success:  true,
		},
		{
			name:     "success: normal case without quotes",
			input:    `PRETTY_NAME=Debian GNU/Linux 12 (bookworm)`,
			expected: "Debian GNU/Linux 12 (bookworm)",
			success:  true,
		},
		{
			name:     "success: with extra lines",
			input:    "PRETTY_NAME=\"Fedora Linux 39\"\nNAME=\"Fedora Linux\"\n",
			expected: "Fedora Linux 39",
			success:  true,
		},
		{
			name:     "failure: no PRETTY_NAME",
			input:    `NAME="Ubuntu"\nVERSION="22.04.3 LTS"`,
			expected: "Invalid",
			success:  false,
		},
		{
			name:     "failure: empty input",
			input:    "",
			expected: "Invalid",
			success:  false,
		},
		{
			name:     "failure: empty first line",
			input:    "\nPRETTY_NAME=\"Ubuntu 22.04.3 LTS\"",
			expected: "Invalid",
			success:  false,
		},
		{
			name:     "failure: PRETTY_NAME not at start",
			input:    "NAME=\"Ubuntu\"\nPRETTY_NAME=\"Ubuntu 22.04.3 LTS\"",
			expected: "Invalid",
			success:  false,
		},
		{
			name:     "success: quotes only",
			input:    `PRETTY_NAME=""`,
			expected: "",
			success:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readDistroFromData([]byte(tt.input))
			if got != tt.expected {
				t.Errorf("readDistroFromData() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// ===================== readDistro =====================
func TestReadDistro(t *testing.T) {
	oldReadFile := ReadFile
	defer func() { ReadFile = oldReadFile }()

	tests := []struct {
		name        string
		mockData    []byte
		mockErr     error
		wantContain string
		wantError   bool
		success     bool
	}{
		{
			name:        "success: valid os-release",
			mockData:    []byte(`PRETTY_NAME="Ubuntu 22.04.3 LTS"`),
			mockErr:     nil,
			wantContain: "Ubuntu 22.04.3 LTS",
			wantError:   false,
			success:     true,
		},
		{
			name:        "failure: file read error",
			mockData:    nil,
			mockErr:     errors.New("permission denied"),
			wantContain: "Error reading /etc/os-release",
			wantError:   true,
			success:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ReadFile = func(path string) ([]byte, error) {
				if path != "/etc/os-release" {
					return nil, errors.New("unexpected path")
				}
				return tt.mockData, tt.mockErr
			}
			got := readDistro()
			if tt.wantError && !strings.Contains(got, "Error reading") {
				t.Errorf("readDistro() = %q, want error message", got)
			}
			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("readDistro() = %q, want to contain %q", got, tt.wantContain)
			}
		})
	}
	ReadFile = oldReadFile
}

// ===================== readDevicesFromOutput =====================
func TestReadDevicesFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   []byte
		err      error
		expected string
		success  bool
	}{
		{
			name:     "success: normal output",
			output:   []byte("00:00.0 Host bridge: Intel Corporation Device\n00:02.0 VGA compatible controller: Intel Corporation Device"),
			err:      nil,
			expected: "00:00.0 Host bridge: Intel Corporation Device\n00:02.0 VGA compatible controller: Intel Corporation Device",
			success:  true,
		},
		{
			name:     "success: empty output",
			output:   []byte(""),
			err:      nil,
			expected: "",
			success:  true,
		},
		{
			name:     "failure: command error",
			output:   nil,
			err:      errors.New("executable file not found"),
			expected: "Executing lspci failed:",
			success:  false,
		},
		{
			name:     "failure: command error with message",
			output:   []byte("some output"),
			err:      errors.New("command failed"),
			expected: "Executing lspci failed:command failed",
			success:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readDevicesFromOutput(tt.output, tt.err)
			if !strings.Contains(got, tt.expected) && got != tt.expected {
				t.Errorf("readDevicesFromOutput() = %q, want to contain or equal %q", got, tt.expected)
			}
		})
	}
}

// ===================== readDevices =====================
func TestReadDevices(t *testing.T) {
	oldExec := ExecOutput
	defer func() { ExecOutput = oldExec }()

	tests := []struct {
		name        string
		mockOut     []byte
		mockErr     error
		wantContain string
		wantError   bool
		success     bool
	}{
		{
			name:        "success: lspci output",
			mockOut:     []byte("00:00.0 Host bridge: Intel Corporation Device"),
			mockErr:     nil,
			wantContain: "00:00.0 Host bridge",
			wantError:   false,
			success:     true,
		},
		{
			name:        "failure: lspci execution fails",
			mockOut:     nil,
			mockErr:     errors.New("exec failed"),
			wantContain: "Executing lspci failed",
			wantError:   true,
			success:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ExecOutput = func(name string, _ ...string) ([]byte, error) {
				if name == "lspci" {
					return tt.mockOut, tt.mockErr
				}
				return nil, nil
			}
			got := readDevices()
			if tt.wantError && !strings.Contains(got, "Executing lspci failed") {
				t.Errorf("readDevices() = %q, want error message", got)
			}
			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("readDevices() = %q, want to contain %q", got, tt.wantContain)
			}
		})
	}
	ExecOutput = oldExec
}

// ===================== runCommand =====================
func TestRunCommand(t *testing.T) {
	oldRun := RunBashCommand
	defer func() { RunBashCommand = oldRun }()

	tests := []struct {
		name    string
		cmd     string
		mockErr error
		wantErr bool
		wantMsg string
		success bool
	}{
		{
			name:    "success: command succeeds",
			cmd:     "echo ok",
			mockErr: nil,
			wantErr: false,
			success: true,
		},
		{
			name:    "failure: command fails",
			cmd:     "false",
			mockErr: fmt.Errorf("command failed: false\nOutput:\nexit 1"),
			wantErr: true,
			wantMsg: "command failed",
			success: false,
		},
		{
			name:    "failure: invalid command",
			cmd:     "nonexistent_command_xyz123",
			mockErr: fmt.Errorf("command failed: nonexistent_command_xyz123\nOutput:\n..."),
			wantErr: true,
			wantMsg: "command failed",
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RunBashCommand = func(cmd string) error {
				if cmd != tt.cmd {
					return nil
				}
				return tt.mockErr
			}
			err := runCommand(tt.cmd)
			if tt.wantErr {
				if err == nil {
					t.Error("runCommand() expected error, got nil")
					return
				}
				if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
					t.Errorf("runCommand() error = %v, want to contain %q", err, tt.wantMsg)
				}
			} else {
				if err != nil {
					t.Errorf("runCommand() unexpected error: %v", err)
				}
			}
		})
	}
	RunBashCommand = oldRun
}

// ===================== runCommands / diskCommands =====================

func TestRunDiskProcedure_CommandsOrder(t *testing.T) {
	oldRun := RunBashCommand
	defer func() { RunBashCommand = oldRun }()

	var got []string
	RunBashCommand = func(cmd string) error {
		got = append(got, cmd)
		return nil
	}

	err := runCommands(diskCommands())
	if err != nil {
		t.Fatalf("runCommands(diskCommands()): %v", err)
	}

	want := diskCommands()
	if len(got) != len(want) {
		t.Fatalf("commands count: got %d, want %d", len(got), len(want))
	}
	for i, cmd := range want {
		if i >= len(got) || got[i] != cmd {
			t.Errorf("command[%d]: got %q, want %q", i, got[i], cmd)
		}
	}
}

func TestRunDiskProcedure_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name          string
		failingCmd    string
		expectedError string
		success       bool
	}{
		{
			name:          "failure: error on first command (mkdir)",
			failingCmd:    "mkdir -p ~/file_systems_test",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on fallocate",
			failingCmd:    "fallocate",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on mkfs",
			failingCmd:    "mkfs.ext4",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on mount",
			failingCmd:    "sudo mount",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on last command",
			failingCmd:    "rm -rf ~/file_systems_test",
			expectedError: "command failed",
			success:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRun := RunBashCommand
			defer func() { RunBashCommand = oldRun }()
			RunBashCommand = func(cmd string) error {
				if strings.Contains(cmd, tt.failingCmd) {
					return fmt.Errorf("%s: %s", tt.expectedError, cmd)
				}
				return nil
			}
			err := runCommands(diskCommands())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.expectedError)
			}
		})
	}
}

func TestRunDiskProcedure_EarlyExitOnError(t *testing.T) {
	oldRun := RunBashCommand
	defer func() { RunBashCommand = oldRun }()

	var executed []string
	failingIndex := 3
	RunBashCommand = func(cmd string) error {
		executed = append(executed, cmd)
		if len(executed) == failingIndex+1 {
			return fmt.Errorf("command failed: %s", cmd)
		}
		return nil
	}

	commands := diskCommands()
	err := runCommands(commands)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(executed) != failingIndex+1 {
		t.Errorf("expected %d commands executed, got %d", failingIndex+1, len(executed))
	}
}

// ===================== lvmCommands / innerLVMProcedure =====================
func TestRunLVMProcedure_CommandsOrder(t *testing.T) {
	oldRun := RunBashCommand
	defer func() { RunBashCommand = oldRun }()

	var got []string
	RunBashCommand = func(cmd string) error {
		got = append(got, cmd)
		return nil
	}

	homeDir := "/home/test"
	loopDevice := "/dev/loop0"
	commands := lvmCommands(homeDir, loopDevice)
	err := runCommands(commands)
	if err != nil {
		t.Fatalf("runCommands(lvmCommands(...)): %v", err)
	}

	want := lvmCommands(homeDir, loopDevice)
	if len(got) != len(want) {
		t.Fatalf("commands count: got %d, want %d", len(got), len(want))
	}
	for i, cmd := range want {
		if i >= len(got) || got[i] != cmd {
			t.Errorf("command[%d]: got %q, want %q", i, got[i], cmd)
		}
	}
}

func TestRunLVMProcedure_ErrorPropagation(t *testing.T) {
	homeDir := "/home/test"
	loopDevice := "/dev/loop0"
	commands := lvmCommands(homeDir, loopDevice)

	tests := []struct {
		name          string
		failingCmd    string
		expectedError string
		success       bool
	}{
		{
			name:          "failure: error on first command (mkdir)",
			failingCmd:    "mkdir -p",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on losetup",
			failingCmd:    "sudo losetup",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on pvcreate",
			failingCmd:    "sudo pvcreate",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on vgcreate",
			failingCmd:    "sudo vgcreate",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on lvcreate",
			failingCmd:    "sudo lvcreate",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on mkfs",
			failingCmd:    "sudo mkfs.ext4",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on mount",
			failingCmd:    "sudo mount",
			expectedError: "command failed",
			success:       false,
		},
		{
			name:          "failure: error on last command",
			failingCmd:    "sudo rm -rf /mnt/lvm1",
			expectedError: "command failed",
			success:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRun := RunBashCommand
			defer func() { RunBashCommand = oldRun }()
			RunBashCommand = func(cmd string) error {
				if strings.Contains(cmd, tt.failingCmd) {
					return fmt.Errorf("%s: %s", tt.expectedError, cmd)
				}
				return nil
			}
			err := runCommands(commands)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.expectedError)
			}
		})
	}
}

func TestRunLVMProcedure_EarlyExitOnError(t *testing.T) {
	oldRun := RunBashCommand
	defer func() { RunBashCommand = oldRun }()

	var executed []string
	failingIndex := 4
	RunBashCommand = func(cmd string) error {
		executed = append(executed, cmd)
		if len(executed) == failingIndex+1 {
			return fmt.Errorf("command failed: %s", cmd)
		}
		return nil
	}

	homeDir := "/home/test"
	loopDevice := "/dev/loop0"
	commands := lvmCommands(homeDir, loopDevice)
	err := runCommands(commands)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(executed) != failingIndex+1 {
		t.Errorf("expected %d commands executed, got %d", failingIndex+1, len(executed))
	}
}

// ===================== main (flag parsing only) =====================

func TestMain_FlagParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantLVM bool
		wantErr bool
		success bool
	}{
		{
			name:    "success: -lvm flag set",
			args:    []string{"-lvm"},
			wantLVM: true,
			wantErr: false,
			success: true,
		},
		{
			name:    "success: no flags (default)",
			args:    []string{},
			wantLVM: false,
			wantErr: false,
			success: true,
		},
		{
			name:    "failure: invalid flag",
			args:    []string{"-unknown-flag"},
			wantLVM: false,
			wantErr: true,
			success: false,
		},
		{
			name:    "failure: invalid format for int flag",
			args:    []string{"-test-int=not-a-number"},
			wantErr: true,
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			useLVM := fs.Bool("lvm", false, "Use LVM procedure")
			if tt.name == "failure: invalid format for int flag" {
				fs.Int("test-int", 0, "Test int flag")
			}

			err := fs.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && *useLVM != tt.wantLVM {
				t.Errorf("useLVM = %v, want %v", *useLVM, tt.wantLVM)
			}
		})
	}
}
