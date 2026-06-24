.PHONY: all dev dev-frontend dev-backend build clean setup build-frontend build-all-platforms

BINARY_NAME := gowfm
FRONTEND_DIR := frontend
BACKEND_DIR := backend
BUILD_DIR := dist
GO := go
NPM := npm

GIT_TAG := $(shell git describe --tags --always --dirty 2>/dev/null || echo "untagged")
VERSION := $(GIT_TAG)
LDFLAGS := -s -w -X 'goWFM/config.Version=$(VERSION)'

ARGS ?=

all: build

dev: dev-frontend dev-backend

dev-frontend:
	$(NPM) --prefix $(FRONTEND_DIR) run dev

dev-backend:
	@echo "==> Starting Go backend with args: $(ARGS)"
	touch  $(BACKEND_DIR)/internal/web-dist/.keep
	$(GO) -C $(BACKEND_DIR) run . $(ARGS)

build: build-frontend build-binary

build-frontend:
	$(NPM) --prefix $(FRONTEND_DIR) install
	$(NPM) --prefix $(FRONTEND_DIR) run build

build-binary:
	CGO_ENABLED=0 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME) .

# CI/CD 专用：一键编译 3 大平台、2 种架构（共 6 个产品）并自动封装
build-all-platforms:
	mkdir -p $(BUILD_DIR)
	# Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=386 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BUILD_DIR)/$(BINARY_NAME)-linux-386 .
	tar -czf $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64
	tar -czf $(BUILD_DIR)/$(BINARY_NAME)-linux-386.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-386
	
	# Windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=386 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BUILD_DIR)/$(BINARY_NAME)-windows-386.exe .
	zip -qj $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.zip $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	zip -qj $(BUILD_DIR)/$(BINARY_NAME)-windows-386.zip $(BUILD_DIR)/$(BINARY_NAME)-windows-386.exe
	
	# macOS
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	tar -czf $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-amd64
	tar -czf $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64
	
	# 清理未压缩的原始二进制临时文件
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-linux-386
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-windows-386.exe
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64

setup:
	$(NPM) --prefix $(FRONTEND_DIR) install

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux $(BINARY_NAME)-darwin $(BINARY_NAME).exe
	rm -rf $(BACKEND_DIR)/internal/web-dist/*
	rm -f gowfm.db gowfm.db-shm gowfm.db-wal

clean-dist:
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux $(BINARY_NAME)-darwin $(BINARY_NAME).exe
	rm -rf $(BACKEND_DIR)/internal/web-dist/*
