FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=direct GONOSUMDB=* GONOSUMCHECK=* GOINSECURE=*
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" -o oauth4os ./cmd/proxy

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/oauth4os /oauth4os
COPY --from=builder /app/web /web
COPY config.yaml /etc/oauth4os/config.yaml
EXPOSE 8443
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -qO- http://localhost:8443/health || exit 1
ENTRYPOINT ["/oauth4os", "-config", "/etc/oauth4os/config.yaml"]
