VERSION = $(shell git tag --points-at HEAD)
COMMIT = $(shell git rev-parse --short HEAD)

build:
	@cd cmd && go build -o groove -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" .

run:
	@go run cmd/main.go

test:
	go test ./... -race

docker-build:
	docker build -t groove_server .

rebuild:
	docker compose rm -sf server && docker compose up -d --build --no-deps server

remove-images:
	docker rmi $(docker images -f dangling=true -q | tail -n +2)

.PHONY: run test rebuild remove-images