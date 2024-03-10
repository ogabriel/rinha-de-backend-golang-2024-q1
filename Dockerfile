FROM golang:1.22.1-alpine3.19 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .

FROM golang:1.22.1-alpine3.19 AS migrate

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM alpine:3.19 AS release

WORKDIR /app

RUN apk add --no-cache make postgresql-client

COPY Makefile ./
COPY docker-entrypoint.sh ./
COPY migrations ./migrations/
COPY --from=migrate /go/bin/migrate ./
COPY --from=builder /app/rinha-de-backend-golang ./

ENTRYPOINT ["/app/docker-entrypoint.sh"]
