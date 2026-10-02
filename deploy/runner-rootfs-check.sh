#!/bin/sh
# runner-rootfs-check.sh <rootfs dir>: the runner image's content gate (m3-15; t3 §6.1; CI's
# runner-image-acceptance lane runs it over `docker export` of the built image). The image must
# carry no setuid/setgid file, no secret and nothing from the private eval pack.
#   - setuid/setgid: any regular file with mode bits 04000 or 02000;
#   - secrets: a private-key PEM block, or a GitHub / AWS / Slack / generic bearer token shape, in
#     any regular file — except the Go toolchain's own public test certificates (listed below);
#   - pack content: any path naming evalpack or xlearn-evalpack.
set -eu
root=${1:?usage: runner-rootfs-check.sh <rootfs dir>}
fail=0

suid=$(find "$root" -xdev -type f \( -perm -4000 -o -perm -2000 \) | sed "s|^$root||")
if [ -n "$suid" ]; then
	echo "setuid/setgid files:"; echo "$suid"; fail=1
fi

# Public test material the Go toolchain ships outside testdata/ and *_test.go (not secrets).
allow='^/opt/xl/go/src/(net/http/internal/testcert/testcert\.go|crypto/x509/platform_root_key\.pem)$'
secrets=$(grep -rlIE -e '-----BEGIN ([A-Z]+ )?PRIVATE KEY-----' -e 'gh[pousr]_[A-Za-z0-9]{36}' -e 'github_pat_[A-Za-z0-9_]{40,}' \
	-e 'AKIA[0-9A-Z]{16}' -e 'xox[abpr]-[A-Za-z0-9-]{10,}' -e 'aib_[A-Za-z0-9]{16,}' "$root" 2>/dev/null | sed "s|^$root||" | grep -vE "$allow" || true)
if [ -n "$secrets" ]; then
	echo "files that look like they hold a secret:"; echo "$secrets"; fail=1
fi

pack=$(find "$root" -xdev \( -iname '*evalpack*' \) | sed "s|^$root||")
if [ -n "$pack" ]; then
	echo "eval-pack paths:"; echo "$pack"; fail=1
fi

if [ "$fail" -ne 0 ]; then
	exit 1
fi
echo "rootfs check: no setuid/setgid file, no secret, no eval-pack path ($(find "$root" -xdev -type f | wc -l) files)"
