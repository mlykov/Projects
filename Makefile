BINARY_NAME=pod_linux
IMAGE_NAME=mlykov/linux-pod
IMAGE_TAG=latest

.PHONY: build image clean

build:
	@echo "Building Go binary for Linux..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) .
	@echo "Binary built: $(BINARY_NAME)"

image:
	@echo "Building Docker image..."
	@docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

