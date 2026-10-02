# Section C (syscalls), Python: arg names one call from t3 §2.4 A1's dangerous set (or clone with
# a namespace flag), made through ctypes' libc syscall(). The exec filter must kill it with
# SIGSYS (if ctypes' own setup is already outside the python@3.13 allowlist, the kill comes there,
# before the call: the stderr marker tells them apart). "survived: ..." on fd 4 means the filter
# let it through (a failure, whatever the call then returned). Numbers are per arch (amd64,
# arm64); the arch comes from the interpreter's build, not from a syscall.
import sys

NRS = {
    "io_uring_setup": (425, 425),
    "bpf": (321, 280),
    "perf_event_open": (298, 241),
    "userfaultfd": (323, 282),
    "keyctl": (250, 219),
    "add_key": (248, 217),
    "ptrace": (101, 117),
    "process_vm_readv": (310, 270),
    "mount": (165, 40),
    "unshare_newuser": (272, 97),
    "setns": (308, 268),
    "clone_newns": (56, 220),
    "socket_netlink": (41, 198),
}


class Solution:
    def probe(self, arg: str) -> str:
        if arg not in NRS:
            return "bad probe " + arg
        arm64 = sys.implementation._multiarch.startswith("aarch64")
        nr = NRS[arg][1 if arm64 else 0]
        sys.stderr.write("loading ctypes\n")
        sys.stderr.flush()
        import ctypes

        libc = ctypes.CDLL(None, use_errno=True)
        libc.syscall.restype = ctypes.c_long
        buf = ctypes.create_string_buffer(512)
        p = ctypes.addressof(buf)
        L = ctypes.c_long
        args = {
            "io_uring_setup": (1, p),
            "bpf": (0, p, 72),
            "perf_event_open": (p, 0, -1, -1, 0),
            "userfaultfd": (0x80000,),
            "keyctl": (0, -4),
            "add_key": (ctypes.c_char_p(b"user"), ctypes.c_char_p(b"xl-probe"), ctypes.c_char_p(b"x"), 1, -2),
            "ptrace": (0,),
            "process_vm_readv": (0, 0, 0, 0, 0, 0),
            "mount": (ctypes.c_char_p(b"none"), ctypes.c_char_p(b"/w"), ctypes.c_char_p(b"tmpfs"), 0, 0),
            "unshare_newuser": (0x10000000,),
            "setns": (-1, 0),
            "clone_newns": (0x00020000 | 17, 0, 0, 0, 0),
            "socket_netlink": (16, 3, 0),
        }[arg]
        args = tuple(a if isinstance(a, ctypes.c_char_p) else L(a) for a in args)
        sys.stderr.write("calling " + arg + "\n")
        sys.stderr.flush()
        rc = libc.syscall(L(nr), *args)
        return "survived: %s rc=%d errno=%d" % (arg, rc, ctypes.get_errno())
