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
                    # Install required tools (make, wget, etc.)
                    echo "Installing required tools..."
                    if [ "$(id -u)" = "0" ]; then
                        apt-get update -qq && apt-get install -y -qq make wget >/dev/null 2>&1
                    else
                        sudo apt-get update -qq && sudo apt-get install -y -qq make wget >/dev/null 2>&1 || \
                        (sudo apt-get update && sudo apt-get install -y make wget)
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
                '''
            }
        }
        
        stage('Format Check') {
            steps {
                sh '''
                    echo "=== Running Format Check ==="
                    make fmt-check
                '''
            }
        }
        
        stage('Lint') {
            steps {
                sh '''
                    echo "=== Running Linter ==="
                    make lint
                '''
            }
        }
        
        stage('Unit Tests') {
            steps {
                sh '''
                    echo "=== Running Unit Tests ==="
                    make test-ci
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
