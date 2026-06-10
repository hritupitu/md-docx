BIN     := docx2pdf
VERSION := 1.0.0
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install clean release

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) .

install: build
	cp $(BIN) /usr/local/bin/$(BIN)
	@echo "Installed to /usr/local/bin/$(BIN)"

# Cross-compile targets
release:
	GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-darwin-arm64  .
	GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-darwin-amd64  .
	GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-linux-amd64   .
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-windows-amd64.exe .
	@echo "Binaries written to dist/"

clean:
	rm -f $(BIN)
	rm -rf dist/
