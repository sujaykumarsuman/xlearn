# xLearn identity image. A static, CGO-free Go binary on distroless static
# (nonroot, read-only-rootfs friendly). Unlike the gateway it serves no SPA, so
# there is no web build stage — goose migrations are embedded in the binary.
#
#   docker build -f deploy/identity.Dockerfile --build-arg VERSION=$(git describe --tags) \
#     -t xlearn-identity:dev .

FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags="-s -w -X github.com/sujaykumarsuman/xlearn/cmd/identity.version=${VERSION}" \
      -o /out/identity ./cmd/identity

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/identity /usr/local/bin/identity
ENV PORT=8081 \
    LOG_LEVEL=info
USER nonroot:nonroot
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/identity"]
