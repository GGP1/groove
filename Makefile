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

docker-run:
	docker run \ 
	-e SV_HOST="0.0.0.0" -e SV_SSL_CERTFILE="/certs/server.crt" -e SV_SSL_KEYFILE="/certs/server.key" \
	-v "./server/certs/:/certs/" \
	groove_server

rebuild-server:
	docker compose rm -sf server && docker compose up -d --build --no-deps server && docker compose logs -f

remove-images:
	docker rmi $(docker images -f dangling=true -q | tail -n +2)

.PHONY: run test rebuild-server remove-images