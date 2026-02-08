BINARY_NAME=pod_linux
IMAGE_NAME=mlykov/linux-pod
IMAGE_TAG=latest
TEST_IMAGE_NAME=$(IMAGE_NAME):test
PLATFORMS=linux/amd64,linux/arm64

.PHONY: build image image-multi clean test test-integration lint fmt fmt-check

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

# Unit tests (mocked I/O, no privileges). Integration tests excluded by build tag.
test:
	@echo "Building test Docker image..."
	@docker build -f Dockerfile.test -t $(TEST_IMAGE_NAME) . >/dev/null 2>&1
	@echo "Running unit tests (no privileges, mocks used)..."
	@test_output=$$(docker run --rm -v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		go test -json ./... 2>&1); \
	test_exit=$$?; \
	failed_tests=$$(echo "$$test_output" | grep -E '"Action":"fail"' | \
		grep -oE '"Test":"[^"]*"' | sed 's/"Test":"\([^"]*\)"/\1/' | sort -u); \
	if [ $$test_exit -eq 0 ]; then \
		echo "Success: All unit tests passed!"; \
	else \
		echo "Failure: Following tests failed:"; \
		if [ -n "$$failed_tests" ]; then \
			echo "$$failed_tests" | sed 's/^/  - /'; \
		fi; \
		exit 1; \
	fi

# Integration tests: build test binary and run in privileged container (real I/O, LVM/disk).
test-integration:
	@echo "Building test Docker image..."
	@docker build -f Dockerfile.test -t $(TEST_IMAGE_NAME) . >/dev/null 2>&1
	@echo "Building integration test binary and running in privileged container..."
	@test_output=$$(docker run --rm --privileged --cap-add=SYS_ADMIN --cap-add=MKNOD \
		-v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		sh -c 'go test -tags=integration -c -o integration.test . && ./integration.test -test.v -test.run Integration' 2>&1); \
	test_exit=$$?; \
	if [ $$test_exit -eq 0 ]; then \
		echo "Success: Integration tests passed!"; \
	else \
		echo "Failure: Integration test run failed:"; \
		echo "$$test_output" | tail -50; \
		exit 1; \
	fi

fmt:
	@echo "Formatting code with gofmt..."
	@docker build -f Dockerfile.test -t $(TEST_IMAGE_NAME) . >/dev/null 2>&1
	@docker run --rm -v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		gofmt -w . >/dev/null 2>&1
	@echo "Code formatted successfully!"

fmt-check:
	@echo "Checking code formatting with gofmt..."
	@docker build -f Dockerfile.test -t $(TEST_IMAGE_NAME) . >/dev/null 2>&1
	@unformatted=$$(docker run --rm -v $$(pwd):/app -w /app $(TEST_IMAGE_NAME) \
		gofmt -l . 2>&1); \
	if [ -n "$$unformatted" ]; then \
		echo "Error: The following files are not properly formatted:"; \
		echo "$$unformatted" | sed 's/^/  - /'; \
		echo "Run 'make fmt' to fix formatting"; \
		exit 1; \
	fi; \
	echo "Success: All files are properly formatted!"

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
