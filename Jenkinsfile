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
                    # Install or update Go if needed
                    if ! command -v go &> /dev/null || [ "$(go version | grep -o 'go[0-9.]*' | sed 's/go//')" != "1.22" ]; then
                        echo "Installing/Updating Go 1.22..."
                        wget -q https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
                        sudo rm -rf /usr/local/go
                        sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
                        rm -f go1.22.0.linux-amd64.tar.gz
                        export PATH=$PATH:/usr/local/go/bin
                    fi
                    echo "Go version:"
                    /usr/local/go/bin/go version || go version
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
