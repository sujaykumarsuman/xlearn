# xlearn-runner image (m3-15; t3 §6.1; ADR-0030, ADR-0034 §1.5). Built ONLY by
# .github/workflows/runner-release.yml on a `runner-v*` tag, never by deploy.yml.
#
# Reproducible: two clean builds of one commit give the same OCI manifest digest when they use
# the pinned BuildKit image and output settings (CI `runner-repro`; docs/git-strategy.md "Runner
# stream"). The rules that make it so:
#   - every version lives in the ARG pins below (Renovate's regex manager reads them, mi-11);
#   - apt reads snapshot.debian.org at DEBIAN_SNAPSHOT, so package versions are fixed by date;
#   - the Go toolchain is the checksummed tarball, picked by TARGETARCH;
#   - the rootfs stage deletes caches/logs/docs, strips every setuid/setgid bit and sets every
#     mtime to SOURCE_DATE_EPOCH, except GOROOT (a fixed past GOROOT_MTIME, set before the seed is
#     built) and the Go cache seed (its fixed FUTURE mtime, t3 §6.1); no BuildKit
#     rewrite-timestamp (it would clamp the seed's future mtime);
#   - the final image is ONE layer (FROM scratch + COPY of the rootfs) with no wall-clock label.
#
# Local build (the release's exact settings; the builder must be the pinned BuildKit image):
#   docker buildx create --name xl-repro --driver docker-container \
#     --driver-opt image=moby/buildkit:v0.33.1@sha256:cec9f139f45e93c5c69c60f8b07cfad9f43f4ef6b6a6cd917527fea5ff2e3dea
#   docker buildx build --builder xl-repro --no-cache -f deploy/runner.Dockerfile --platform linux/amd64 \
#     --build-arg VERSION=runner-vX.Y.Z --build-arg REVISION=$(git rev-parse HEAD) \
#     --build-arg SOURCE_DATE_EPOCH=$(git log -1 --format=%ct) --provenance=false --sbom=false \
#     --output type=oci,dest=runner.oci.tar,oci-mediatypes=true,compression=gzip,compression-level=9,force-compression=true .

# ---- pins (the only place versions live) ----
# The multi-arch INDEX digest: compose (arm64 Macs), the arm64 VM rehearsal and the amd64 release
# build the same file.
ARG BASE=debian:trixie-slim@sha256:a99cfc517144bc59b1978475ec53b46ecabec7e43635402ee5b77cc54cd1b20a
ARG GO_BUILD_IMAGE=golang:1.26.8-trixie@sha256:eae2aaa6add2936cbf350dd0d2628b363461542f0c4b3c0b558957e0f2997379
ARG DEBIAN_SNAPSHOT=20261001T000000Z
ARG GO_VERSION=1.26.8
ARG GO_SHA256_AMD64=d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b
ARG GO_SHA256_ARM64=211ffced9dcb9633a55eac6364816ec0ddd951389a740e88fa8b3337971bdda0

# ---- build: the runner binary (static, world-executable; m3-03's hand-off) ----
# Cross-compiled on the build platform: a CGO-free -trimpath Go build is byte-identical whatever
# the host arch, so an arm64 Mac and the amd64 CI runner produce the same binary.
FROM --platform=$BUILDPLATFORM ${GO_BUILD_IMAGE} AS build
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=${VERSION}" -o /out/runner ./cmd/runner

# ---- gotar: the pinned Go toolchain from its checksummed tarball, pruned ----
FROM --platform=$BUILDPLATFORM ${GO_BUILD_IMAGE} AS gotar
ARG TARGETARCH
ARG GO_VERSION
ARG GO_SHA256_AMD64
ARG GO_SHA256_ARM64
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]
RUN <<'EOF'
case "${TARGETARCH}" in
  amd64) sum="${GO_SHA256_AMD64}" ;;
  arm64) sum="${GO_SHA256_ARM64}" ;;
  *) echo "unsupported TARGETARCH ${TARGETARCH}" >&2; exit 1 ;;
esac
tgz="go${GO_VERSION}.linux-${TARGETARCH}.tar.gz"
curl -fsSL --retry 5 --retry-all-errors -o "/tmp/${tgz}" "https://go.dev/dl/${tgz}"
echo "${sum}  /tmp/${tgz}" | sha256sum -c -
mkdir -p /opt/xl
tar -C /opt/xl -xzf "/tmp/${tgz}"
# Pruned (t3 §6.1): nothing a `go build` of learner code or `go build std` reads.
rm -rf /opt/xl/go/test /opt/xl/go/api /opt/xl/go/doc /opt/xl/go/misc
find /opt/xl/go/src -depth \( -name testdata -type d -exec rm -rf {} + \) -o \( -name '*_test.go' -type f -delete \)
EOF

# ---- rootfs: Debian trixie (snapshot) + g++ 14 + CPython 3.13 + Go + the runner ----
FROM ${BASE} AS rootfs
ARG DEBIAN_SNAPSHOT
ARG SOURCE_DATE_EPOCH
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]
# apt from snapshot.debian.org (http: the keyring signs the Release files; the slim base has no CA
# bundle), so a package version is a function of DEBIAN_SNAPSHOT.
RUN <<'EOF'
cat > /etc/apt/sources.list.d/debian.sources <<SOURCES
Types: deb
URIs: http://snapshot.debian.org/archive/debian/${DEBIAN_SNAPSHOT}
Suites: trixie trixie-updates
Components: main
Signed-By: /usr/share/keyrings/debian-archive-keyring.pgp
Check-Valid-Until: no

Types: deb
URIs: http://snapshot.debian.org/archive/debian-security/${DEBIAN_SNAPSHOT}
Suites: trixie-security
Components: main
Signed-By: /usr/share/keyrings/debian-archive-keyring.pgp
Check-Valid-Until: no
SOURCES
apt_opts=(-o Acquire::Retries=8 -o Acquire::http::Timeout=120 -o Acquire::Check-Valid-Until=false)
apt-get "${apt_opts[@]}" update
DEBIAN_FRONTEND=noninteractive apt-get "${apt_opts[@]}" -y upgrade
DEBIAN_FRONTEND=noninteractive apt-get "${apt_opts[@]}" -y install --no-install-recommends g++ python3
g++ --version | head -1
python3 -VV
EOF
COPY --from=gotar /opt/xl/go /opt/xl/go
COPY --from=build /out/runner /usr/local/bin/runner
# The go@1.26 GOCACHE seed (m3-04's recipe, `gocache-seed@1`): `go build std` with the profile's
# exact toolchain, flags and env; a non-zero exit fails the build; the tree hash goes to the log.
# GOROOT gets a FIXED past mtime first, never SOURCE_DATE_EPOCH: the go command's module index
# (cached in GOCACHE) keys each std package on its files' mtimes, so the seed only hits when the
# image's GOROOT mtimes equal the seed build's; a commit-independent mtime also keeps the seed's
# tree hash, and so go@1.26's profile_sha256 (the calibration key), unchanged across runner tags.
ARG GOROOT_MTIME=2000-01-01T00:00:00Z
RUN <<'EOF'
find /opt/xl/go -exec touch -h -d "${GOROOT_MTIME}" {} +
echo "gocache seed tree hash: $(/usr/local/bin/runner seed-gocache -out /opt/xl/gocache)"
EOF
RUN <<'EOF'
# Python: the stdlib bytecode as checked-hash .pyc (valid whatever the mtime; python3 -B never
# writes one). Debian's postinst wrote timestamp .pyc that the mtime normalization would stale.
find /usr/lib/python3* -name __pycache__ -type d -prune -exec rm -rf {} +
python3 -m compileall -q -j 1 --invalidation-mode checked-hash /usr/lib/python3.13
# Mount points the spawner uses under a read-only root: /jail is the pod's emptyDir (the only
# place the AppArmor profile admits mounts).
install -d -m 0755 /jail
# Strip every setuid/setgid bit (t3 §6.1).
find / -xdev -type f -perm /6000 -exec chmod ug-s {} +
# Caches, logs, docs, man pages, apt lists, and the files that record when they were written.
rm -rf /var/lib/apt/lists/* /var/cache/apt/* /var/cache/debconf/*-old /var/lib/dpkg/*-old \
  /var/cache/ldconfig/aux-cache /var/log/* /usr/share/doc/* /usr/share/man/* /usr/share/info/* \
  /tmp/* /var/tmp/* /root/.cache /root/.wget-hsts
# Every mtime to SOURCE_DATE_EPOCH (directories last-modified by the deletes above included),
# except GOROOT (its fixed GOROOT_MTIME, above) and the Go cache seed (its fixed FUTURE mtime,
# goprofile.SeedTime, re-asserted here). BuildKit's read-only bind mounts (/etc/hosts,
# /etc/resolv.conf, /etc/hostname) are not part of the layer.
test -n "${SOURCE_DATE_EPOCH}"
find / -xdev \( -path /proc -o -path /sys -o -path /dev -o -path /etc/hosts -o -path /etc/resolv.conf \
  -o -path /etc/hostname -o -path /opt/xl/go -o -path /opt/xl/gocache \) -prune \
  -o -exec touch -h -d "@${SOURCE_DATE_EPOCH}" {} +
find /opt/xl/go -exec touch -h -d "${GOROOT_MTIME}" {} +
find /opt/xl/gocache -exec touch -h -d "2100-01-01T00:00:00Z" {} +
if find / -xdev -type f -perm /6000 | grep -q .; then echo "setuid/setgid file left" >&2; exit 1; fi
EOF

# ---- final: one layer ----
FROM scratch
ARG VERSION=dev
ARG REVISION=unknown
COPY --from=rootfs / /
ENV PATH=/usr/local/bin:/usr/bin:/bin
LABEL org.opencontainers.image.source="https://github.com/sujaykumarsuman/xlearn" \
      org.opencontainers.image.title="xlearn-runner" \
      org.opencontainers.image.description="xLearn runner: the sandbox that runs learner code for judge (ADR-0030)" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/runner"]
