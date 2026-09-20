# xLearn curriculum image. A static, CGO-free Go binary on distroless static
# (nonroot, read-only-rootfs friendly). Like identity it serves no SPA, so there is
# no web build stage — goose migrations AND the versioned curriculum/ seed files are
# embedded in the binary (the whole repo, incl. curriculum/, is copied in below).
#
#   docker build -f deploy/curriculum.Dockerfile --build-arg VERSION=$(git describe --tags) \
#     -t xlearn-curriculum:dev .

FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags="-s -w -X github.com/sujaykumarsuman/xlearn/cmd/curriculum.version=${VERSION}" \
      -o /out/curriculum ./cmd/curriculum

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/curriculum /usr/local/bin/curriculum
ENV PORT=8082 \
    LOG_LEVEL=info
USER nonroot:nonroot
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/curriculum"]
