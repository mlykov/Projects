package main

import (
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

	fmt.Printf("Produced output:\n%s\n", string(out))

	if err != nil {
		return fmt.Errorf(
			"Command failed: %s\nOutput:\n%s",
			command,
			string(out),
		)
	}

	return nil
}

func runDiskProcedure() error {
	fmt.Println("=== Running Disk Procedure ===")

	fmt.Println("Executing: mkdir -p ~/file_systems_test")
	if err := runCommand("mkdir -p ~/file_systems_test"); err != nil {
		return err
	}

	fmt.Println("Executing: cd ~/file_systems_test")
	if err := runCommand("cd ~/file_systems_test"); err != nil {
		return err
	}

	fmt.Println("Executing: fallocate -l 100M disk1")
	if err := runCommand("fallocate -l 100M disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: mkfs.ext4 -F disk1")
	if err := runCommand("mkfs.ext4 -F disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: mkdir -p /mnt/disk1")
	if err := runCommand("sudo mkdir -p /mnt/disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: sudo mount -o loop disk1 /mnt/disk1")
	if err := runCommand("sudo mount -o loop disk1 /mnt/disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: sudo bash -c 'printf Hello ext4 > /mnt/disk1/test.txt'")
	if err := runCommand(`sudo bash -c 'printf "Hello ext4\n" > /mnt/disk1/test.txt'`); err != nil {
		return err
	}

	fmt.Println("Executing: cat /mnt/disk1/test.txt")
	if err := runCommand(`cat /mnt/disk1/test.txt`); err != nil {
		return err
	}

	fmt.Println("Executing: sudo umount /mnt/disk1")
	if err := runCommand("sudo umount /mnt/disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: wipefs -a disk1")
	if err := runCommand("wipefs -a disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: rm -f disk1")
	if err := runCommand("rm -f disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: sudo rm -rf /mnt/disk1")
	if err := runCommand("sudo rm -rf /mnt/disk1"); err != nil {
		return err
	}

	fmt.Println("Executing: rm -rf ~/file_systems_test")
	if err := runCommand("rm -rf ~/file_systems_test"); err != nil {
		return err
	}

	return nil
}

func main() {
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
		fmt.Println("------------------------\nDisk procedure: ")

		if err := runDiskProcedure(); err != nil {
			fmt.Println("ERROR in disk procedure:")
			fmt.Println(err)
		} else {
			fmt.Println("------------------------\nDisk procedure: Success")
		}

		time.Sleep(15 * time.Second)
	}
}
