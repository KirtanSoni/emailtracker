BINARY_NAME=readtracker
GO=go

.PHONY: build clean

build:
	$(GO) mod tidy
	$(GO) build -o $(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME) emails.db go.sum