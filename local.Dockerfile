# Build the manager binary
FROM golang:1.25 AS builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
COPY cmd cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o exporter cmd/main.go
RUN chmod +x exporter

# Use distroless as minimal base image to package the application
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/exporter /exporter
EXPOSE 9100
ENTRYPOINT ["/exporter"]