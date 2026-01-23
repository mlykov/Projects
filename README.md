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

## Verify Multi-Platform Build

```bash
docker manifest inspect mlykov/linux-pod:latest
```

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

### Build Commands

- `make build` - (optional) compiles Go application into `pod_linux` binary. Not required for Docker usage as the image build process compiles automatically.
- `make image` - builds Docker image for current platform (includes automatic compilation of Go application). Use for local testing.
- `make image-multi` - builds and pushes multi-platform Docker image to Docker Hub for `linux/amd64` and `linux/arm64` architectures
- `make clean` - removes compiled binary and test Docker images

### Testing and Code Quality

- `make test` - runs all unit and integration tests in Docker. Tests are executed in a privileged container with required capabilities for LVM operations. Outputs a summary:
  - `Success: All tests passed!` - if all tests pass
  - `Failure: Following tests failed:` - with a list of failed tests if any fail

- `make lint` - checks code style and quality in Docker:
  - Verifies code formatting with `gofmt`
  - Runs static analysis with `go vet`
  - Outputs:
    - `Success: All lint checks passed!` - if all checks pass
    - `Failure: Lint checks failed!` - with details about formatting issues or vet errors

- `make fmt` - automatically formats all Go code using `gofmt` in Docker. Use this to fix formatting issues before committing code.

**Note:** All testing and linting commands run in Docker to ensure consistency across different systems and to provide the Linux environment required for integration tests.

## Continuous Integration

This project uses GitHub Actions for continuous integration. The CI pipeline runs automatically on:
- Pull requests to `main` or `master` branches
- Pushes to `main` or `master` branches

The CI pipeline includes three parallel jobs:
1. **test** - Runs all unit and integration tests (`make test`)
2. **lint** - Checks code style and quality (`make lint`)
3. **fmt-check** - Verifies that all code is properly formatted

All checks must pass before a pull request can be merged.

## Jenkins CI/CD

This project also includes Jenkins pipeline configuration for continuous integration.

### Quick Start

1. **Start Jenkins:**
   ```bash
   docker-compose up -d
   ```

2. **Get initial password:**
   ```bash
   docker exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword
   ```

3. **Access Jenkins:**
   - Open http://localhost:8080 in your browser
   - Enter the initial password
   - Install recommended plugins
   - Create an admin user

4. **Create Pipeline Job:**
   - Click "New Item" → "Pipeline"
   - Configure: Pipeline script from SCM → Git → Repository URL
   - Script Path: `Jenkinsfile`

5. **Run Pipeline:**
   - Click "Build Now" to execute the pipeline

### Jenkins Pipeline Stages

The Jenkins pipeline (`Jenkinsfile`) executes:
1. **Checkout** - Clones the repository
2. **Setup Go** - Installs/updates Go 1.22
3. **Format Check** - Runs `make fmt-check`
4. **Lint** - Runs `make lint`
5. **Unit Tests** - Runs `make test-ci`

## Project Structure
```
linux-pod/
├── main.go          # Application source code
├── main_test.go     # Unit and integration tests
├── go.mod           # Go dependencies
├── Dockerfile       # Docker image for running the application
├── Dockerfile.test  # Docker image for running tests and linting
├── pod.yaml         # Kubernetes manifest (default mode)
├── pod-lvm.yaml     # Kubernetes manifest (LVM mode)
├── Makefile         # Build, test, and lint commands
├── Jenkinsfile      # Jenkins CI/CD pipeline
├── docker-compose.yml # Docker Compose for Jenkins
├── .github/
│   └── workflows/
│       └── ci.yml   # GitHub Actions CI workflow
├── JENKINS_SETUP.md # Jenkins setup instructions
└── README.md        # Documentation
```