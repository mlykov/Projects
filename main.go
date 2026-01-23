package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func readCpuCores() int {
	cpu_cores := 0

	out, err := exec.Command("nproc").Output()

	if err != nil {
		fmt.Println("Executing nproc failed:", err)
		return 0
	}

	cpu_cores, err = strconv.Atoi(strings.TrimSpace(string(out)))

	if err != nil {
		fmt.Println("Parsing nproc output failed:", err)
		return 0
	}

	return cpu_cores
}

func readMemoryFromData(data []byte) (usedKB, freeKB float64) {
	lines := strings.Split(string(data), "\n")

	if len(lines) < 3 {
		fmt.Println("/proc/meminfo output is not defined as expected - not enough lines")
		return 0, 0
	}

	// Parse MemTotal from first line
	parts := strings.Fields(lines[0])
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "MemTotal:") {
		fmt.Println("/proc/meminfo output is not defined as expected - MemTotal: is not on right line")
		return 0, 0
	}

	val, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("Error parsing amount of KB in MemTotal:", err)
		return 0, 0
	}
	total := float64(val)

	// Parse MemAvailable from third line (index 2)
	parts = strings.Fields(lines[2])
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "MemAvailable:") {
		fmt.Println("/proc/meminfo output is not defined as expected - MemAvailable: is not on right line")
		return 0, 0
	}

	val, err = strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("Error parsing amount of KB in MemAvailable:", err)
		return 0, 0
	}
	free := float64(val)

	used := total - free
	return used, free
}

func readMemory() (usedKB, freeKB float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		fmt.Println("Reading /proc/meminfo failed:", err)
		return 0, 0
	}
	return readMemoryFromData(data)
}

func readDistroFromData(data []byte) string {
	lines := strings.Split(string(data), "\n")

	if len(lines) == 0 || lines[0] == "" {
		fmt.Println("/etc/os-release output is not defined as expected")
		return "Invalid"
	}

	if strings.HasPrefix(lines[0], "PRETTY_NAME=") {
		pretty := strings.TrimPrefix(lines[0], "PRETTY_NAME=")
		pretty = strings.Trim(pretty, "\"")
		return pretty
	}

	fmt.Println("/etc/os-release output is not defined as expected")
	return "Invalid"
}

func readDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Error reading /etc/os-release: " + err.Error()
	}
	return readDistroFromData(data)
}

func readDevicesFromOutput(output []byte, err error) string {
	if err != nil {
		return "Executing lspci failed:" + err.Error()
	}
	return string(output)
}

func readDevices() string {
	out, err := exec.Command("lspci").Output()
	return readDevicesFromOutput(out, err)
}

func runCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(
			"command failed: %s\nOutput:\n%s",
			command,
			string(out),
		)
	}

	return nil
}

func runCommands(commands []string, cmdRunner func(string) error) error {
	for _, cmd := range commands {
		fmt.Printf("Executing: %s\n", cmd)
		if err := cmdRunner(cmd); err != nil {
			return err
		}
	}
	return nil
}

func diskCommands() []string {
	return []string{
		"mkdir -p ~/file_systems_test",
		"cd ~/file_systems_test",
		"fallocate -l 100M disk1",
		"mkfs.ext4 -F disk1",
		"sudo mkdir -p /mnt/disk1",
		"sudo mount -o loop disk1 /mnt/disk1",
		`sudo bash -c 'printf "Hello ext4\n" > /mnt/disk1/test.txt'`,
		"cat /mnt/disk1/test.txt",
		"sudo umount /mnt/disk1",
		"wipefs -a disk1",
		"rm -f disk1",
		"sudo rm -rf /mnt/disk1",
		"rm -rf ~/file_systems_test",
	}
}

func runDiskProcedure() error {
	fmt.Println("=== Running Disk Procedure ===")
	return runCommands(diskCommands(), runCommand)
}

// lvmCommands returns the list of commands that runLVMProcedure executes.
// This function is pure and testable without command execution.
func lvmCommands(homeDir, loopDevice string) []string {
	testDir := fmt.Sprintf("%s/file_systems_test", homeDir)
	diskFile := fmt.Sprintf("%s/disk1", testDir)

	return []string{
		fmt.Sprintf("mkdir -p %s", testDir),
		fmt.Sprintf("fallocate -l 100M %s", diskFile),
		fmt.Sprintf("sudo losetup %s %s", loopDevice, diskFile),
		fmt.Sprintf("sudo pvcreate -y %s", loopDevice),
		fmt.Sprintf("sudo vgcreate testvg %s", loopDevice),
		"sudo lvcreate -Z n -l 50%FREE -n testlv1 testvg",
		"sudo lvcreate -Z n -l 100%FREE -n testlv2 testvg",
		"sudo vgchange -ay testvg",
		"sudo vgscan --mknodes",
		"sudo mkfs.ext4 -F /dev/mapper/testvg-testlv1",
		"sudo mkfs.ext4 -F /dev/mapper/testvg-testlv2",
		"sudo mkdir -p /mnt/lvm1 /mnt/lvm2",
		"sudo mount /dev/mapper/testvg-testlv1 /mnt/lvm1",
		"sudo mount /dev/mapper/testvg-testlv2 /mnt/lvm2",
		`sudo bash -c 'printf "Hello LVM LV1\n" > /mnt/lvm1/test.txt'`,
		`sudo bash -c 'printf "Hello LVM LV2\n" > /mnt/lvm2/test.txt'`,
		"cat /mnt/lvm1/test.txt",
		"cat /mnt/lvm2/test.txt",
		"sudo umount /mnt/lvm1 /mnt/lvm2",
		"sudo lvremove -y testvg/testlv1 testvg/testlv2",
		"sudo vgremove -y testvg",
		fmt.Sprintf("sudo pvremove -y %s", loopDevice),
		fmt.Sprintf("sudo losetup -d %s", loopDevice),
		fmt.Sprintf("rm -f %s", diskFile),
		fmt.Sprintf("sudo rm -rf /mnt/lvm1 /mnt/lvm2 %s", testDir),
	}
}

func runLVMProcedureWithDeps(homeDirGetter func() (string, error), loopDeviceGetter func() (string, error), cmdRunner func(string) error) error {
	fmt.Println("=== Running LVM Procedure ===")

	// Get home directory
	homeDir, err := homeDirGetter()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	testDir := fmt.Sprintf("%s/file_systems_test", homeDir)

	// Find first free loop device
	loopDevice, err := loopDeviceGetter()
	if err != nil {
		return fmt.Errorf("failed to find free loop device: %w", err)
	}
	fmt.Printf("Using loop device: %s\n", loopDevice)

	// Cleanup from previous failed runs
	cleanupCommands := []string{
		"sudo umount /mnt/lvm1 /mnt/lvm2 2>/dev/null || true",
		"sudo lvremove -y testvg/testlv1 testvg/testlv2 2>/dev/null || true",
		"sudo vgremove -y testvg 2>/dev/null || true",
		"sudo rm -rf /dev/testvg 2>/dev/null || true",
		fmt.Sprintf("sudo pvremove -y %s 2>/dev/null || true", loopDevice),
		fmt.Sprintf("sudo losetup -d %s 2>/dev/null || true", loopDevice),
		fmt.Sprintf("sudo rm -rf /mnt/lvm1 /mnt/lvm2 %s 2>/dev/null || true", testDir),
	}

	for _, cmd := range cleanupCommands {
		cmdRunner(cmd)
	}

	// Actual LVM procedure
	commands := lvmCommands(homeDir, loopDevice)
	return runCommands(commands, cmdRunner)
}

func runLVMProcedure() error {
	homeDirGetter := func() (string, error) {
		return os.UserHomeDir()
	}
	loopDeviceGetter := func() (string, error) {
		loopDeviceBytes, err := exec.Command("bash", "-c", "sudo losetup -f").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(loopDeviceBytes)), nil
	}
	return runLVMProcedureWithDeps(homeDirGetter, loopDeviceGetter, runCommand)
}

func main() {
	useLVM := flag.Bool("lvm", false, "Use LVM procedure")
	flag.Parse()

	for {
		fmt.Println("=== Mashine Info ===")
		cores := readCpuCores()
		fmt.Printf("CPU cores: %d\n", cores)
		used, free := readMemory()
		fmt.Printf("Used memory: %.2f GB or %.2f MB\n", float64(used)/1024/1024, used/1024)
		fmt.Printf("Free memory: %.2f GB or %.2f MB\n", float64(free)/1024/1024, free/1024)
		distro := readDistro()
		fmt.Printf("Distribution: %s\n", distro)
		divices := readDevices()
		fmt.Printf("Devices:\n%s\n", divices)

		var err error
		if *useLVM {
			err = runLVMProcedure()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("=== LVM Procedure Completed Successfully ===")
				fmt.Println()

			}
		} else {
			err = runDiskProcedure()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("=== Disk Procedure Completed Successfully ===")
				fmt.Println()
			}
		}

		time.Sleep(15 * time.Second)
	}
}
