BIN     := md2docx-gui
LDFLAGS := -s -w

.PHONY: build install release clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) .

install: build
	cp $(BIN) /usr/local/bin/$(BIN)
	@echo "Installed → /usr/local/bin/$(BIN)"

release:
	mkdir -p dist
	GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-macos-arm64  .
	GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-macos-amd64  .
	GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-linux-amd64   .
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-windows-amd64.exe .
	@echo "Binaries in dist/"

clean:
	rm -f $(BIN)
	rm -rf dist/
