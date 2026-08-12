FROM golang:1.25 AS build
ARG VERSION=0.2.0
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X forgejo-bridge/internal/cli.Version=${VERSION} -X forgejo-bridge/internal/cli.Commit=${COMMIT} -X forgejo-bridge/internal/cli.BuildDate=${BUILD_DATE}" \
    -o /out/forgejo-bridge ./cmd/forgejo-bridge

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/forgejo-bridge /forgejo-bridge
USER 65532:65532
ENTRYPOINT ["/forgejo-bridge"]
