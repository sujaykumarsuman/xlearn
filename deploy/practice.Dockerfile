# xLearn practice image. A static, CGO-free Go binary on distroless static
# (nonroot, read-only-rootfs friendly). Like identity/curriculum it serves no SPA, so
# there is no web build stage — goose migrations are embedded in the binary.
#
#   docker build -f deploy/practice.Dockerfile --build-arg VERSION=$(git describe --tags) \
#     -t xlearn-practice:dev .

FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags="-s -w -X github.com/sujaykumarsuman/xlearn/cmd/practice.version=${VERSION}" \
      -o /out/practice ./cmd/practice

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/practice /usr/local/bin/practice
ENV PORT=8083 \
    LOG_LEVEL=info
USER nonroot:nonroot
EXPOSE 8083
ENTRYPOINT ["/usr/local/bin/practice"]
