.PHONY: build install clean rpm

BINARY_NAME=quickplan
VERSION?=0.1.0
BUILD_DIR=build
RPM_DIR=$(BUILD_DIR)/rpm

# Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

# Install to local system (requires sudo)
install: build
	@echo "Installing $(BINARY_NAME)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installed successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

# Build RPM package
rpm: build
	@echo "Building RPM package..."
	@mkdir -p $(RPM_DIR)/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	@mkdir -p $(RPM_DIR)/SOURCES/$(BINARY_NAME)-$(VERSION)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(RPM_DIR)/SOURCES/$(BINARY_NAME)-$(VERSION)/
	@cp quickplan.spec $(RPM_DIR)/SPECS/
	@cd $(RPM_DIR) && rpmbuild --define "_topdir $$(pwd)" \
		--define "version $(VERSION)" \
		-bb SPECS/quickplan.spec
	@echo "RPM built: $(RPM_DIR)/RPMS/*/quickplan-$(VERSION)-1.*.rpm"

# Run the application
run:
	go run . --help

# Show help
help:
	@echo "QuickPlan Makefile commands:"
	@echo "  make build    - Build the binary"
	@echo "  make install  - Install to /usr/local/bin"
	@echo "  make rpm      - Build RPM package"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make run      - Run the application"
