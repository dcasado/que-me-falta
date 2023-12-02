FROM golang:1.21.4-alpine3.17 AS builder

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

# Add group and user
RUN addgroup -S quemefalta && adduser -S quemefalta -G quemefalta

# Build binary
RUN CGO_ENABLED=0 go build -o quemefalta cmd/quemefalta/main.go


FROM scratch

ENV LISTEN_ADDRESS="0.0.0.0"

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/quemefalta /usr/bin/quemefalta

USER quemefalta

EXPOSE 8080

ENTRYPOINT ["/usr/bin/quemefalta"]
