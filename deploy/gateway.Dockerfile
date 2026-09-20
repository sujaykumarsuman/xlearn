# xLearn gateway image. Multi-stage, mirroring the Makefile: build the web SPA
# (Vite -> web/dist), embed it, then a static CGO-free Go binary stamped with
# VERSION, onto distroless static (nonroot, read-only rootfs friendly).
#
#   docker build -f deploy/gateway.Dockerfile --build-arg VERSION=$(git describe --tags) \
#     -t xlearn-gateway:dev .

# --- web build (Vite -> web/dist) ---
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run --silent build

# --- go build (embeds web/dist) ---
FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
# Overwrite the .gitkeep-only web/dist with the built assets go:embed needs.
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags="-s -w -X github.com/sujaykumarsuman/xlearn.Version=${VERSION}" \
      -o /out/gateway ./cmd/gateway

# --- runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gateway /usr/local/bin/gateway
ENV PORT=8080 \
    LOG_LEVEL=info \
    BASE_PATH=/xlearn
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway"]
