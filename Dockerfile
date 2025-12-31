# Use distroless as minimal base image to package the application
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
ARG TARGETPLATFORM
WORKDIR /
COPY $TARGETPLATFORM/exporter /exporter
EXPOSE 9100
ENTRYPOINT ["/exporter"]