# IKEA DIRIGERA Exporter

A Prometheus Metrics Exporter for the IKEA DIRIGERA Smart-Home-Hub. This library is an unofficial implementation
based on the [IKEA DIRIGERA Client](https://github.com/salex-org/ikea-dirigera-client) and is not affiliated with
IKEA of Sweden AB!

The current version is work in progress that does not cover all functions and has not been fully tested.
**Use this exporter at your own risk!**

## Build locally

Build and run locally on MacOS:

```shell
goreleaser release --snapshot --clean
docker run -d -p 9100:9100 ghcr.io/salex-org/ikea-dirigera-exporter:latest-arm64
```

Call metrics endpoint locally:

```shell
curl http://localhost:9100/metrics
```