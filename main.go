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

func readMemory() (usedKB, freeKB float64) {

	data, err := os.ReadFile("/proc/meminfo")

	if err != nil {
		fmt.Println("Reading /proc/meminfo failed:", err)
		return 0, 0
	}

	lines := strings.Split(string(data), "\n")

	parts := strings.Fields(lines[0])

	if !strings.HasPrefix(parts[0], "MemTotal:") {
		fmt.Println("/proc/meminfo output is not defined as expected - MemTotal: is not on right line")
	}

	val, err := strconv.Atoi(parts[1])

	if err != nil {
		fmt.Println("Error parsing amount of KB in MemTotal:", err)
		return 0, 0
	}

	usedKB = float64(val)

	parts = strings.Fields(lines[2])

	if !strings.HasPrefix(parts[0], "MemAvailable:") {
		fmt.Println("/proc/meminfo output is not defined as expected - MemAvailable: is not on right line")
	}

	val, err = strconv.Atoi(parts[1])

	if err != nil {
		fmt.Println("Error parsing amount of KB in MemAvailable:", err)
		return 0, 0
	}

	freeKB = float64(val)

	usedKB = (usedKB - freeKB)

	return usedKB, freeKB
}

func readDistro() string {
	data, err := os.ReadFile("/etc/os-release")

	if err != nil {
		return "Error reading /etc/os-release: " + err.Error()
	}

	lines := strings.Split(string(data), "\n")
	var pretty string = "Invalid"

	if strings.HasPrefix(lines[0], "PRETTY_NAME=") {
		pretty = strings.TrimPrefix(lines[0], "PRETTY_NAME=")
		pretty = strings.Trim(pretty, "\"")

	} else {
		fmt.Println("/etc/os-release output is not defined as expected")
	}

	return pretty
}

func readDevices() string {
	out, err := exec.Command("lspci").Output()

	if err != nil {
		return "Executing lspci failed:" + err.Error()
	}

	return string(out)
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

func runDiskProcedure() error {
	fmt.Println("=== Running Disk Procedure ===")

	commands := []string{
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

	for _, cmd := range commands {
		fmt.Printf("Executing: %s\n", cmd)
		if err := runCommand(cmd); err != nil {
			return err
		}
	}

	return nil
}

func runLVMProcedure() error {
	fmt.Println("=== Running LVM Procedure ===")

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	testDir := fmt.Sprintf("%s/file_systems_test", homeDir)
	diskFile := fmt.Sprintf("%s/disk1", testDir)

	// Find first free loop device
	loopDeviceBytes, err := exec.Command("bash", "-c", "sudo losetup -f").Output()
	if err != nil {
		return fmt.Errorf("failed to find free loop device: %w", err)
	}
	loopDevice := strings.TrimSpace(string(loopDeviceBytes))
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
		runCommand(cmd)
	}

	// Actual LVM procedure
	commands := []string{
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

	for _, cmd := range commands {
		fmt.Printf("Executing: %s\n", cmd)
		if err := runCommand(cmd); err != nil {
			return err
		}
	}

	return nil
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
