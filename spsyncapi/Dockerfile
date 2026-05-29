# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/spsyncapi .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget \
	&& adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/spsyncapi /app/spsyncapi
COPY config.docker.yaml /app/config.yaml

RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/spsyncapi"]
CMD ["serve"]
