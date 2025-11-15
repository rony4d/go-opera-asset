# Directory where compiled binaries will be placed.
BIN_DIR := bin
# Full path for the built executable.
BINARY  := $(BIN_DIR)/opera-asset

# Declare phony targets to avoid conflicts with files.
.PHONY: build test run tidy clean

# Build target compiles the project into the bin directory.
build:
# Ensure the bin directory exists before building.
	@mkdir -p $(BIN_DIR)
# Compile the code in ./cmd into the binary path.
	go build -o $(BINARY) ./cmd

# Run target depends on build so the binary is up to date.
run: build
# Execute the compiled binary.
	$(BINARY)

# Test target runs all Go tests in the module.
test:
# Execute go test across all packages.
	go test ./...

# Tidy target synchronizes module definitions.
tidy:
# Run go mod tidy to update go.mod and go.sum.
	go mod tidy

# Clean target removes build outputs and cached artifacts.
clean:
# Remove Go build cache artifacts for the module.
	go clean
# Delete the bin directory with compiled binaries.
	rm -rf $(BIN_DIR)


# Make Commands:
# make build - Build the project into the bin directory.
# make run - Run the compiled binary.
# make test - Run all Go tests in the module.
# make tidy - Synchronize module definitions.
# make clean - Remove build outputs and cached artifacts.