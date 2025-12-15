BINARY_NAME=pod_linux
IMAGE_NAME=mlykov/linux-pod
IMAGE_TAG=latest
PLATFORMS=linux/amd64,linux/arm64

.PHONY: build image image-multi clean

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

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

