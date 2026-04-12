FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=direct GONOSUMDB=* GOFLAGS=-insecure
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -o oauth4os ./cmd/proxy

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/oauth4os /usr/local/bin/oauth4os
COPY config.yaml /etc/oauth4os/config.yaml
EXPOSE 8443
ENTRYPOINT ["oauth4os", "-config", "/etc/oauth4os/config.yaml"]
