# Linux Pod

Go application that outputs system information and performs disk/LVM procedures every 15 seconds to stdout. 

## What the application does

- Outputs CPU cores information
- Shows used and free memory
- Detects Linux distribution
- Lists PCI devices
- Performs disk procedures:
  - **Default mode** (without flags): Creates ext4 file system on a loop device, mounts it, writes/reads test files, then cleans up
  - **LVM mode** (`-lvm` flag): Creates LVM setup - splits a disk file into two logical volumes using LVM, formats them, mounts, writes/reads test files, then cleans up
- Updates information in stdout every 15 seconds

## Requirements

- Go 1.22 or higher
- Docker (docker daemon must be running)
- kind - optional, for creating a local Kubernetes cluster
- kubectl - optional, for running in Kubernetes cluster

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/mlykov/Projects.git
cd linux-pod
```

### 2. Build Docker image

```bash
make image
```

This will build the Docker image `mlykov/linux-pod:latest`.

**Note:** The `make build` command before `make image` is optional and only creates a local binary file `pod_linux`. The Docker build process compiles the application automatically, so you don't need to run `make build` separately.

### 3. Run the image

**Option A: Default mode (disk procedure)**

```bash
docker run --rm --privileged mlykov/linux-pod:latest
```

**Option B: LVM mode**

```bash
docker run --rm --privileged mlykov/linux-pod:latest -lvm
```

**Option C: In Kubernetes cluster**

Set up and run the pod using kind:

```bash
# Check existing clusters
kind get clusters

# Delete already existing cluster if needed
kind delete cluster --name linux-pod-cluster
```

```bash
# Create kind cluster
kind create cluster --name linux-pod-cluster --image kindest/node:v1.30.0

# Verify cluster exists
kind get clusters
```

Choose the appropriate manifest file:
- `pod.yaml` - for default mode (disk procedure)
- `pod-lvm.yaml` - for LVM mode

```bash
# Delete existing pod if it exists (required when changing pod configuration)
kubectl delete pod linux-pod

# Apply the manifest - choose one:
# For default mode:
kubectl apply -f pod.yaml

# OR for LVM mode:
kubectl apply -f pod-lvm.yaml

# Check pod status, wait until STATUS becomes RUNNING
kubectl get pods

# View logs
kubectl logs -f linux-pod

```
Clean up when done

```bash
kubectl delete pod linux-pod
kind delete cluster --name linux-pod-cluster
```

## Makefile Commands

- `make build` - (optional) compiles Go application into `pod_linux` binary. Not required for Docker usage as the image build process compiles automatically.
- `make image` - builds Docker image (includes automatic compilation of Go application)
- `make clean` - removes compiled binary

## Project Structure

```
linux-pod/
├── main.go          # Application source code
├── go.mod           # Go dependencies
├── Dockerfile       # Docker image
├── pod.yaml         # Kubernetes manifest (default mode)
├── pod-lvm.yaml     # Kubernetes manifest (LVM mode)
├── Makefile         # Build commands
└── README.md        # Documentation
```