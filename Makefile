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
	@mkdir -p ./.coverage
	@mkdir -p ./.test-results
	@go test -json ./... -cover -coverpkg=./... -coverprofile="./.coverage/cover.out" | go-test-report -o ./.test-results/report.html -g 8
	@go tool cover -html="./.coverage/cover.out" -o "./.coverage/report.html"
	@echo "Test report generated at ./.test-results/report.html"
	@echo "Coverage report generated at ./.coverage/report.html"

# Define the default target
.PHONY: all
all: build