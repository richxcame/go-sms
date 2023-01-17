dev:
	go run main.go

build:
	env GOOS=linux GOARCH=amd64 go build -o bin/gosms
