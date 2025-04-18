FROM golang:1.18.3-alpine3.16 as builder

RUN apk add --update --no-cache ca-certificates && update-ca-certificates

WORKDIR /go/src/app

COPY go.mod .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o groove -ldflags="-s -w" ./cmd/main.go

FROM scratch

COPY --from=builder /go/src/app/groove /usr/bin/groove

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/usr/bin/groove"]