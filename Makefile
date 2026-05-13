.PHONY: all dev dev-frontend dev-backend build clean setup

BINARY_NAME := gowfm
FRONTEND_DIR := frontend
BACKEND_DIR := backend
GO := go
NPM := npm

GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG := $(shell git describe --tags --always --dirty 2>/dev/null || echo "untagged")
VERSION := $(GIT_BRANCH)-$(GIT_TAG)
LDFLAGS := -s -w -X 'goWFM.Version=$(VERSION)'

all: build

dev: dev-frontend dev-backend

dev-frontend:
	$(NPM) --prefix $(FRONTEND_DIR) run dev

dev-backend:
	$(GO) -C $(BACKEND_DIR) run .

build: build-frontend build-binary

build-frontend:
	$(NPM) --prefix $(FRONTEND_DIR) install
	$(NPM) --prefix $(FRONTEND_DIR) run build

build-binary: build-frontend
	CGO_ENABLED=0 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME) .

build-linux: build-frontend
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME)-linux .

build-darwin: build-frontend
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME)-darwin .

build-windows: build-frontend
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) -C $(BACKEND_DIR) build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME).exe .

setup:
	$(NPM) --prefix $(FRONTEND_DIR) install

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux $(BINARY_NAME)-darwin $(BINARY_NAME).exe
	rm -rf $(BACKEND_DIR)/internal/web-dist/*
	rm -f gowfm.db gowfm.db-shm gowfm.db-wal

clean-dist:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux $(BINARY_NAME)-darwin $(BINARY_NAME).exe
	rm -rf $(BACKEND_DIR)/internal/web-dist/*
