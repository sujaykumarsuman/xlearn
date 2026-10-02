# Section E (cross-job markers, t3 §5.10), Python: arg is "<write|read>:<channel>:<token>". A write
# answers "written" or "denied: ..." (or dies of SIGSYS: the call, or the module it needs, is
# outside python@3.13's exec allowlist); a read answers "visible" or "absent: ...". Job N's
# markers must be invisible to job N+1.
import os

DIRS = {"w": "/w/", "tmp": "/tmp/", "devshm": "/dev/shm/"}


def fnv(s):
    h = 2166136261
    for c in s.encode():
        h = ((h ^ c) * 16777619) & 0xFFFFFFFF
    return h & 0x7FFFFFFF


class Solution:
    def probe(self, arg: str) -> str:
        parts = arg.split(":", 2)
        if len(parts) != 3:
            return "bad probe " + arg
        op, ch, tok = parts
        name = "xl-" + tok
        write = op == "write"
        if ch in DIRS:
            path = DIRS[ch] + name
            if write:
                try:
                    with open(path, "w") as f:
                        f.write(tok)
                except OSError as e:
                    return "denied: " + repr(e)
                return "written"
            try:
                os.close(os.open(path, os.O_RDONLY))
            except OSError as e:
                return "absent: " + repr(e)
            return "visible"
        if ch == "abstract":
            import socket

            try:
                s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
                if write:
                    s.bind("\0" + name)
                    s.listen(1)
                    return "written"
                s.connect("\0" + name)
                return "visible"
            except OSError as e:
                return ("denied: " if write else "absent: ") + repr(e)
        import ctypes

        libc = ctypes.CDLL(None, use_errno=True)
        if ch == "sysvshm":
            if write:
                rc = libc.shmget(fnv(tok), 4096, 0o1000 | 0o666)
                return "denied: errno %d" % ctypes.get_errno() if rc < 0 else "written"
            rc = libc.shmget(fnv(tok), 0, 0)
            return "absent: errno %d" % ctypes.get_errno() if rc < 0 else "visible"
        if ch == "posixmq":
            q = ("/" + name).encode()
            if write:
                rc = libc.mq_open(q, os.O_CREAT | os.O_RDWR, 0o666, None)
                return "denied: errno %d" % ctypes.get_errno() if rc < 0 else "written"
            rc = libc.mq_open(q, os.O_RDONLY)
            return "absent: errno %d" % ctypes.get_errno() if rc < 0 else "visible"
        if ch == "keyring":
            libc.syscall.restype = ctypes.c_long
            import sys

            arm64 = sys.implementation._multiarch.startswith("aarch64")
            if write:
                rc = libc.syscall(ctypes.c_long(217 if arm64 else 248), b"user", name.encode(), tok.encode(),
                                  ctypes.c_long(len(tok)), ctypes.c_long(-4))
                return "denied: errno %d" % ctypes.get_errno() if rc < 0 else "written"
            rc = libc.syscall(ctypes.c_long(219 if arm64 else 250), ctypes.c_long(10), ctypes.c_long(-4), b"user",
                              name.encode(), ctypes.c_long(0))
            return "absent: errno %d" % ctypes.get_errno() if rc < 0 else "visible"
        return "bad probe " + arg
