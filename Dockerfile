FROM golang:1.23.2-alpine3.20 AS builder
WORKDIR /build
COPY go.mod .
COPY go.sum .
# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
# Build
COPY . .
RUN go build .

FROM alpine:latest
USER root
ENV USER=exporter
ENV GROUPNAME=$USER
ENV UID=12001
ENV GID=12001
COPY --from=builder /build/prometheus-aws-cost-exporter /usr/bin
RUN addgroup \
    --gid "$GID" \
    "$GROUPNAME" \
&& adduser \
    --disabled-password \
    --gecos "" \
    --home "$(pwd)" \
    --ingroup "$GROUPNAME" \
    --no-create-home \
    --uid "$UID" \
    $USER
USER exporter:exporter