FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=direct GONOSUMDB=* GOFLAGS=-insecure
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" -o oauth4os ./cmd/proxy

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/oauth4os /oauth4os
COPY --from=builder /app/web /web
COPY config.yaml /etc/oauth4os/config.yaml
EXPOSE 8443
ENTRYPOINT ["/oauth4os", "-config", "/etc/oauth4os/config.yaml"]
