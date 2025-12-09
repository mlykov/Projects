BINARY_NAME=pod_linux
BINARY_NAME_MAC=pod_mac
IMAGE_NAME=mlykov/linux-pod
IMAGE_TAG=latest

.PHONY: build build-mac image clean

build:
	@echo "Building Go binary for Linux..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) .
	@echo "Binary built: $(BINARY_NAME)"

build-mac:
	@echo "Building Go binary for macOS..."
	@CGO_ENABLED=0 go build -o $(BINARY_NAME_MAC) .
	@echo "Binary built: $(BINARY_NAME_MAC)"

image: build
	@echo "Building Docker image..."
	@docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME_MAC)
	@echo "Clean complete"

