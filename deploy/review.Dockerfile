# xLearn review image. A static, CGO-free Go binary on distroless static
# (nonroot, read-only-rootfs friendly). Like identity/curriculum/practice it serves no
# SPA, so there is no web build stage — goose migrations are embedded in the binary.
#
#   docker build -f deploy/review.Dockerfile --build-arg VERSION=$(git describe --tags) \
#     -t xlearn-review:dev .

FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/review ./cmd/review

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/review /usr/local/bin/review
ENV PORT=8084 \
    LOG_LEVEL=info
USER nonroot:nonroot
EXPOSE 8084
ENTRYPOINT ["/usr/local/bin/review"]
