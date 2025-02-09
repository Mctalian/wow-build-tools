# Define the output directory and binary name
OUTPUT_DIR := dist
BINARY_NAME := wow-build-tools

# Define the build command
build:
	mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(BINARY_NAME)

# Define the clean command
clean:
	rm -rf $(OUTPUT_DIR)

# Define the run command
run: build
	./$(OUTPUT_DIR)/$(BINARY_NAME)

test:
	@go test -v ./... -cover -coverpkg=./... -coverprofile="./.coverage/cover.out"
	@go tool cover -html="./.coverage/cover.out" -o "./.coverage/report.html"
	@echo "Coverage report generated at ./.coverage/report.html"

# Define the default target
.PHONY: all
all: build