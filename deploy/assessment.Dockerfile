# xLearn assessment image. A static, CGO-free Go binary on distroless static
# (nonroot, read-only-rootfs friendly). Like identity/curriculum/practice/review it
# serves no SPA, so there is no web build stage — goose migrations are embedded in the
# binary.
#
#   docker build -f deploy/assessment.Dockerfile --build-arg VERSION=$(git describe --tags) \
#     -t xlearn-assessment:dev .

FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags="-s -w -X github.com/sujaykumarsuman/xlearn/cmd/assessment.version=${VERSION}" \
      -o /out/assessment ./cmd/assessment

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/assessment /usr/local/bin/assessment
ENV PORT=8085 \
    LOG_LEVEL=info
USER nonroot:nonroot
EXPOSE 8085
ENTRYPOINT ["/usr/local/bin/assessment"]
