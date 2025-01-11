.PHONY: compile-receiver
compile-receiver:
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o koreader-customizations/plugins/m4k.koplugin/m4k_receiver cmd/m4k_receiver/main.go

.PHONY: install
install:
	go install ./cmd/m4k

