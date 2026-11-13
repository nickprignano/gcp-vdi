FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/gcp-vdi-broker ./cmd/broker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/gcp-vdi-broker /usr/local/bin/gcp-vdi-broker
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gcp-vdi-broker"]
