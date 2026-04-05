# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies (git is required for go mod download)
RUN apk add --no-cache gcc musl-dev git

WORKDIR /app

# Set Go module proxy for reliable downloads
# Uses Google proxy with direct fallback; switch to goproxy.cn if in China mainland
ENV GOPROXY=https://goproxy.io,https://proxy.golang.org,direct
ENV GONOSUMCHECK=*
ENV GOFLAGS=-mod=mod

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with retry logic
RUN go mod download || go mod download || go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# ========================
# Runtime stage
FROM alpine:3.19

# Install runtime dependencies:
#   ca-certificates  - TLS/SSL connections to cloud platform APIs
#   tzdata           - timezone support (Asia/Shanghai)
#   bind-tools       - DNS diagnostic tools (dig, nslookup, host)
#                      for debugging domain-based cloud platform access
#   curl             - HTTP diagnostic tool for API connectivity testing
RUN apk --no-cache add ca-certificates tzdata bind-tools curl

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/server .

# ========================
# /etc/hosts Mount for Domain-based Cloud Platform Access
# ========================
# When connecting to cloud platforms (EasyStack/ZStack) that use domain
# names resolved via the host's /etc/hosts (private infrastructure
# without public DNS), the host's /etc/hosts file must be mounted into
# this container as a read-only volume.
#
# This is configured in docker-compose.yml:
#   volumes:
#     - /etc/hosts:/etc/hosts:ro
#
# Alternatively, use extra_hosts in docker-compose.yml:
#   extra_hosts:
#     - "keystone.easystack.local:10.0.0.100"
#     - "zstack.internal.corp:10.0.0.200"
#
# To verify DNS resolution inside the container:
#   docker exec cloud-agent-backend nslookup keystone.easystack.local
#   docker exec cloud-agent-backend cat /etc/hosts
# ========================

EXPOSE 8080

CMD ["./server"]
