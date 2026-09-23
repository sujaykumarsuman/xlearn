# xLearn coach image. A static, CGO-free Go binary on distroless static (nonroot,
# read-only-rootfs friendly). Like identity/curriculum/practice/review/assessment it
# serves no SPA, so there is no web build stage — goose migrations are embedded in the
# binary. The envelope master key is mounted at runtime via the xlearn-coach secret
# (COACH_MASTER_KEY); it is never baked into the image.
#
#   docker build -f deploy/coach.Dockerfile --build-arg VERSION=$(git describe --tags) \
#     -t xlearn-coach:dev .

FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/coach ./cmd/coach

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coach /usr/local/bin/coach
ENV PORT=8086 \
    LOG_LEVEL=info
USER nonroot:nonroot
EXPOSE 8086
ENTRYPOINT ["/usr/local/bin/coach"]
