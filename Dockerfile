FROM golang:1.21.4-alpine3.17 AS builder

# Install build packages
RUN apk add --no-cache \
  # Important: required for go-sqlite3
  gcc=12.2.1_git20220924-r4 \
  # Required for Alpine
  musl-dev=1.2.3-r6

# Add group and user
RUN addgroup -S quemefalta && adduser -S quemefalta -G quemefalta

WORKDIR /app

# Copy modules files
COPY go.mod ./
COPY go.sum ./

# Download Go modules
RUN go mod download

# Copy the rest of the files
COPY cmd ./cmd
COPY internal ./internal
COPY models.go ./

# Build binary. cgo needed because of go-sqlite3
RUN CGO_ENABLED=1 go build -o quemefalta -a -ldflags '-extldflags "-static"' cmd/quemefalta/main.go


FROM scratch

ENV DATABASE_URI="/data/quemefalta.db"
ENV LISTEN_ADDRESS="0.0.0.0"

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/quemefalta /usr/bin/quemefalta

USER quemefalta

VOLUME [ "/data" ]

EXPOSE 8080

ENTRYPOINT ["/usr/bin/quemefalta"]
