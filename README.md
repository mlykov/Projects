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

**For local testing:**
```bash
make image
```

This will build the Docker image `mlykov/linux-pod:latest` for your current platform. 
**Note:** The `make build` command before `make image` is optional and only creates a local binary file `pod_linux`. The Docker build process compiles the application automatically, so you don't need to run `make build` separately.

**For multi-platform build:**
```bash
make image-multi
```

This will build and push the multi-platform Docker image `mlykov/linux-pod:latest` for both `linux/amd64` and `linux/arm64` architectures to Docker Hub.

### 3. Run the image

**Option A: Default mode**

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
# Create kind cluster
kind create cluster --name linux-pod-cluster --image kindest/node:v1.30.0

# Verify cluster exists
kind get clusters
```

**Default mode:**
```bash
# Apply the manifest
kubectl apply -f pod.yaml

# Check pod status, wait until STATUS becomes RUNNING
kubectl get pods

# View logs
kubectl logs -f linux-pod
```

**LVM mode:**
```bash
# Apply the LVM manifest
kubectl apply -f pod-lvm.yaml

# Check pod status, wait until STATUS becomes RUNNING
kubectl get pods

# View logs
kubectl logs -f linux-pod
```

Clean up when done:
```bash
kubectl delete pod linux-pod
kind delete cluster --name linux-pod-cluster
```

## Makefile Commands

- `make build` - (optional) compiles Go application into `pod_linux` binary. Not required for Docker usage as the image build process compiles automatically.
- `make image` - builds Docker image for current platform (includes automatic compilation of Go application). Use for local testing.
- `make image-multi` - builds and pushes multi-platform Docker image to Docker Hub for `linux/amd64` and `linux/arm64` architectures
- `make clean` - removes compiled binary

## Verifying Multi-Platform Build

```bash
docker manifest inspect mlykov/linux-pod:latest
```

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