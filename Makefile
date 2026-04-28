.PHONY: all dev dev-frontend dev-backend build clean setup

BINARY_NAME := wfm
FRONTEND_DIR := frontend
BUILD_DIR := build
GO := go
NPM := npm

all: build

dev: dev-frontend dev-backend

dev-frontend:
	$(NPM) --prefix $(FRONTEND_DIR) run dev

dev-backend:
	$(GO) run .

build: build-frontend build-binary

build-frontend:
	$(NPM) --prefix $(FRONTEND_DIR) install
	$(NPM) --prefix $(FRONTEND_DIR) run build

build-binary: build-frontend
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BINARY_NAME) .

build-linux: build-frontend
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(BINARY_NAME) .

build-darwin: build-frontend
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="-s -w" -o $(BINARY_NAME) .

build-windows: build-frontend
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(BINARY_NAME).exe .

setup:
	$(NPM) --prefix $(FRONTEND_DIR) install

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf $(FRONTEND_DIR)/dist
	rm -f wfm.db wfm.db-shm wfm.db-wal

clean-dist:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf $(FRONTEND_DIR)/dist
