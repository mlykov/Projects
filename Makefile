BINARY_NAME=pod_linux
IMAGE_NAME=mlykov/linux-pod
IMAGE_TAG=latest
TEST_IMAGE_NAME=$(IMAGE_NAME):test
PLATFORMS=linux/amd64,linux/arm64

.PHONY: build image image-multi clean test lint fmt

build:
	@echo "Building Go binary for current platform..."
	@CGO_ENABLED=0 go build -o $(BINARY_NAME) .
	@echo "Binary built: $(BINARY_NAME)"

image:
	@echo "Building Docker image for current platform..."
	@docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

image-multi:
	@echo "Setting up Docker buildx..."
	@docker buildx create --name multiarch --use 2>/dev/null || docker buildx use multiarch
	@docker buildx inspect --bootstrap
	@echo "Building and pushing multi-platform Docker image for $(PLATFORMS)..."
	@docker buildx build --platform $(PLATFORMS) -t $(IMAGE_NAME):$(IMAGE_TAG) --push .
	@echo "Multi-platform image built and pushed: $(IMAGE_NAME):$(IMAGE_TAG)"

test:
	@echo "Building test Docker image..."
	@docker build -f Dockerfile.test -t $(TEST_IMAGE_NAME) . >/dev/null 2>&1
	@echo "Running tests in Docker..."
	@test_output=$$(docker run --rm --privileged --cap-add=SYS_ADMIN --cap-add=MKNOD \
		-v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		go test -json ./... 2>&1); \
	test_exit=$$?; \
	failed_tests=$$(echo "$$test_output" | grep -E '"Action":"fail"' | \
		grep -oE '"Test":"[^"]*"' | sed 's/"Test":"\([^"]*\)"/\1/' | sort -u); \
	if [ $$test_exit -eq 0 ]; then \
		echo "Success: All tests passed!"; \
	else \
		echo "Failure: Following tests failed:"; \
		if [ -n "$$failed_tests" ]; then \
			echo "$$failed_tests" | sed 's/^/  - /'; \
		fi; \
		exit 1; \
	fi

fmt:
	@echo "Formatting code with gofmt..."
	@docker build -f Dockerfile.test -t $(TEST_IMAGE_NAME) . >/dev/null 2>&1
	@docker run --rm -v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		gofmt -w . >/dev/null 2>&1
	@echo "Code formatted successfully!"

lint:
	@echo "Building test Docker image..."
	@docker build -f Dockerfile.test -t $(TEST_IMAGE_NAME) . >/dev/null 2>&1
	@echo "Running linters in Docker..."
	@lint_errors=0; \
	echo "Checking code formatting with gofmt..."; \
	unformatted=$$(docker run --rm -v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		gofmt -l . 2>&1); \
	if [ -n "$$unformatted" ]; then \
		echo "Error: The following files are not properly formatted:"; \
		echo "$$unformatted" | sed 's/^/  - /'; \
		echo "Run 'make fmt' to fix formatting"; \
		lint_errors=1; \
	fi; \
	echo "Running go vet..."; \
	if ! docker run --rm -v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		go vet ./... 2>&1; then \
		lint_errors=1; \
	fi; \
	if [ $$lint_errors -eq 0 ]; then \
		echo "Success: All lint checks passed!"; \
	else \
		echo "Failure: Lint checks failed!"; \
		exit 1; \
	fi

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@docker rmi $(TEST_IMAGE_NAME) >/dev/null 2>&1 || true
	@echo "Clean complete"
