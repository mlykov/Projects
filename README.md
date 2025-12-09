# Linux Pod

Go application that outputs system information and performs disk procedures every 15 seconds to stdout. 

## What the application does

- Outputs CPU cores information
- Shows used and free memory
- Detects Linux distribution
- Lists PCI devices
- Performs disk procedures (creates ext4 file system, mounting, writing/reading files)
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

### 2. Build the binary

```bash
make build
```

This will create a binary file `pod_linux` in the current directory.

### 3. Build Docker image

```bash
make image
```

This will build the Docker image `mlykov/linux-pod:latest`.

### 4. Run the image

**Option A: Locally via Docker**

```bash
docker run --rm --privileged mlykov/linux-pod:latest
```

**Option B: In Kubernetes cluster**

```bash
# Create kind cluster
kind create cluster --name linux-pod-cluster

# Load image into cluster
kind load docker-image mlykov/linux-pod:latest --name linux-pod-cluster

# Apply manifest
kubectl apply -f pod.yaml

# View logs
kubectl logs -f linux-pod
```

## Makefile Commands

- `make build` - compiles Go application into `pod_linux` binary
- `make image` - builds Docker image 
- `make clean` - removes compiled binary

## Project Structure

```
linux-pod/
├── main.go          # Application source code
├── go.mod           # Go dependencies
├── Dockerfile       # Docker image
├── pod.yaml         # Kubernetes manifest
├── Makefile         # Build commands
└── README.md        # Documentation
```

## Management

```bash
# Delete pod
kubectl delete pod linux-pod

# Delete cluster
kind delete cluster --name linux-pod-cluster
```
