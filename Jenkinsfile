pipeline {
    agent any
    
    environment {
        DOCKER_IMAGE = 'mlykov/linux-pod:test'
        PATH = "/usr/local/go/bin:${env.PATH}"
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Setup Go') {
            steps {
                sh '''
                    # Install required tools (make, wget, docker, etc.)
                    echo "Installing required tools..."
                    if [ "$(id -u)" = "0" ]; then
                        apt-get update -qq && apt-get install -y -qq make wget docker.io >/dev/null 2>&1 || \
                        apt-get update && apt-get install -y make wget docker.io
                    else
                        sudo apt-get update -qq && sudo apt-get install -y -qq make wget docker.io >/dev/null 2>&1 || \
                        (sudo apt-get update && sudo apt-get install -y make wget docker.io)
                    fi
                    
                    # Verify Docker is accessible
                    if ! docker ps >/dev/null 2>&1; then
                        echo "Warning: Docker may not be accessible. Checking Docker socket..."
                        ls -la /var/run/docker.sock || echo "Docker socket not found"
                    fi
                    
                    # Install or update Go if needed
                    if ! command -v go &> /dev/null || [ "$(go version 2>/dev/null | grep -o 'go[0-9.]*' | sed 's/go//' || echo '0')" != "1.22" ]; then
                        echo "Installing/Updating Go 1.22..."
                        # Download Go
                        if command -v wget &> /dev/null; then
                            wget -q https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
                        elif command -v curl &> /dev/null; then
                            curl -L -o go1.22.0.linux-amd64.tar.gz https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
                        else
                            echo "Error: Neither wget nor curl is available"
                            exit 1
                        fi
                        # Install Go
                        if [ "$(id -u)" = "0" ]; then
                            rm -rf /usr/local/go
                            tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
                        else
                            sudo rm -rf /usr/local/go
                            sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
                        fi
                        rm -f go1.22.0.linux-amd64.tar.gz
                        export PATH=$PATH:/usr/local/go/bin
                    fi
                    echo "Go version:"
                    /usr/local/go/bin/go version || go version
                    echo "Make version:"
                    make --version
                    echo "Docker version:"
                    docker --version || echo "Docker not available"
                '''
            }
        }
        
        stage('Format Check') {
            steps {
                sh '''
                    echo "=== Running Format Check ==="
                    # Check formatting directly with gofmt (no Docker needed)
                    unformatted=$(gofmt -l . 2>&1)
                    if [ -n "$unformatted" ]; then
                        echo "Error: The following files are not properly formatted:"
                        echo "$unformatted" | sed 's/^/  - /'
                        echo "Run 'make fmt' to fix formatting"
                        exit 1
                    fi
                    echo "Success: All files are properly formatted!"
                '''
            }
        }
        
        stage('Lint') {
            steps {
                sh '''
                    echo "=== Running Linter ==="
                    # Run lint checks directly (no Docker needed)
                    lint_errors=0
                    echo "Checking code formatting with gofmt..."
                    unformatted=$(gofmt -l . 2>&1)
                    if [ -n "$unformatted" ]; then
                        echo "Error: The following files are not properly formatted:"
                        echo "$unformatted" | sed 's/^/  - /'
                        echo "Run 'make fmt' to fix formatting"
                        lint_errors=1
                    fi
                    echo "Running go vet..."
                    if ! go vet ./... 2>&1; then
                        lint_errors=1
                    fi
                    if [ $lint_errors -eq 0 ]; then
                        echo "Success: All lint checks passed!"
                    else
                        echo "Failure: Lint checks failed!"
                        exit 1
                    fi
                '''
            }
        }
        
        stage('Unit Tests') {
            steps {
                sh '''
                    echo "=== Running Unit Tests ==="
                    # Run tests directly (no Docker needed, but skip integration tests)
                    go test -short -v ./... 2>&1 | tee test_output.txt
                    test_exit=${PIPESTATUS[0]}
                    if [ $test_exit -eq 0 ]; then
                        echo "Success: All tests passed!"
                    else
                        echo "Failure: Some tests failed!"
                        exit 1
                    fi
                '''
            }
        }
    }
    
    post {
        success {
            echo 'Success: All checks passed successfully!'
        }
        failure {
            echo 'Failure: Some checks failed. Please review the logs.'
        }
    }
}
