.PHONY: build clean test install

PLUGIN_NAME=observability.so
BUILD_DIR=../../build/plugins
INSTALL_DIR=~/.s9s/plugins

# Build the plugin as a shared object
build:
	@echo "Building observability plugin..."
	@mkdir -p $(BUILD_DIR)
	cd cmd/plugin && go build -buildmode=plugin -o ../../$(BUILD_DIR)/$(PLUGIN_NAME) .

# Install the plugin to the user's plugin directory
install: build
	@echo "Installing observability plugin..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(PLUGIN_NAME) $(INSTALL_DIR)/
	@echo "Plugin installed to $(INSTALL_DIR)/$(PLUGIN_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BUILD_DIR)/$(PLUGIN_NAME)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Build for development (in the local plugins directory)
dev:
	@echo "Building for development..."
	@mkdir -p ./build
	cd cmd/plugin && go build -buildmode=plugin -o ../../build/$(PLUGIN_NAME) .
	@echo "Development plugin built at ./build/$(PLUGIN_NAME)"