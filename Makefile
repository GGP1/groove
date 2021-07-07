VERSION = $(shell git tag --points-at HEAD)
COMMIT = $(shell git rev-parse --short HEAD)

build:
	@cd cmd && go build -o groove -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" .

.PHONY: run
run:
	@go run cmd/main.go

.PHONY: test
test:
	go test ./... -race

.PHONY: docker-build
docker-build:
	docker build -t groove_server .

docker-run:
	docker run \ 
	-e SV_HOST="0.0.0.0" -e SV_SSL_CERTFILE="/certs/server.crt" -e SV_SSL_KEYFILE="/certs/server.key" \
	-v "./server/certs/:/certs/" \
	groove_server

.PHONY: rebuild-server
rebuild-server:
	docker compose rm -sf server && docker compose up --build --no-deps server

.PHONY: remove-images
remove-images:
	docker rmi $(docker images -f dangling=true -q | tail -n +2)