APP_NAME := folder_sync_s3.exe

tidy:
	@echo "tidying up..."
	@go mod tidy
	@go mod vendor

build:
	@echo "building..."
	@go build -o ./bin/${APP_NAME} ./src/main.go

run: tidy build
	@./bin/${APP_NAME}

dev: build
	@./bin/${APP_NAME}
