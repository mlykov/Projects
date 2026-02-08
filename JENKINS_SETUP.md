# Jenkins Setup Guide

This guide provides instructions for setting up and configuring Jenkins CI/CD pipeline for the Linux Pod project.

## Prerequisites

- Docker and Docker Compose installed on your system
- Git repository access (for Jenkins to clone the project)
- Administrative access to configure Jenkins

## Running Jenkins via Docker

### Step 1: Start Jenkins Container

Execute the following command to start the Jenkins container:

```bash
docker-compose up -d
```

This command creates and starts the Jenkins container, making it accessible at `http://localhost:8080`.

### Step 2: Retrieve Initial Administrator Password

Obtain the initial administrator password required for first-time Jenkins setup:

```bash
docker exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword
```

Copy the displayed password and use it during the initial Jenkins login process.

### Step 3: Configure Jenkins

1. Open `http://localhost:8080` in your web browser
2. Enter the initial administrator password obtained in Step 2
3. Install recommended plugins (this includes the Docker Pipeline plugin, which is required)
4. Create an administrative user account

### Step 4: Docker Configuration

The Jenkins container is pre-configured to use Docker through volume mount (`/var/run/docker.sock`). No additional Docker configuration is required.

### Step 5: Create Pipeline Job

1. In the Jenkins Dashboard, click **"New Item"**
2. Select **"Pipeline"** as the job type
3. Enter a job name (e.g., `linux-pod-ci`)
4. Click **"OK"**
5. In the **"Pipeline"** section:
   - **Definition**: Change from "Pipeline script" to **"Pipeline script from SCM"** (this is important)
   - **SCM**: Select **"Git"**
   - **Repository URL**: Enter your repository URL (e.g., `https://github.com/your-username/your-repo.git`)
   - **Credentials**: If the repository is private, add credentials for authentication
   - **Branches to build**: 
     - Click **"Add"** next to "Branch Specifier"
     - Enter `*/main` (if your default branch is `main`)
     - Or enter `*/master` (if your default branch is `master`)
     - **Important**: Ensure the specified branch exists in your repository
   - **Script Path**: Verify that `Jenkinsfile` is specified
6. Click **"Save"**

### Step 6: Execute Pipeline

Click **"Build Now"** to trigger the pipeline execution.

## Pipeline Stages

The Jenkins pipeline (`Jenkinsfile`) executes the following stages in sequence:

1. **Checkout** - Clones the repository from the configured source control
2. **Setup Go** - Installs or updates Go 1.22 and required build tools (make, wget)
3. **Format Check** - Verifies code formatting using `gofmt` (`make fmt-check` equivalent)
4. **Lint** - Performs code quality checks using `gofmt` and `go vet` (`make lint` equivalent)
5. **Unit Tests** - Executes unit tests with integration tests skipped (`make test` equivalent)

## Stopping Jenkins

To stop the Jenkins container:

```bash
docker-compose down
```

To stop and remove all data (including Jenkins configuration):

```bash
docker-compose down -v
```

**Warning**: The second command will delete all Jenkins data, including jobs, configurations, and build history.

## Access Information

- **Jenkins URL**: http://localhost:8080
- **Port 50000**: Used for Jenkins agents (if required)

## Troubleshooting

### Docker Access Issues

If pipeline stages fail with Docker-related errors, verify that:
- Docker socket is properly mounted: `/var/run/docker.sock`
- Jenkins container has appropriate permissions to access Docker

### Build Failures

If builds fail, check:
- Go version is correctly installed (should be 1.22)
- Required tools (make, wget) are available
- Repository URL and branch name are correct
- Jenkinsfile exists in the repository root
