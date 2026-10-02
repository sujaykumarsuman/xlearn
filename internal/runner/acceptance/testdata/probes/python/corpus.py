# Section D (t3 §9's P2 corpus), Python, plus the no-op and spin programs sections A and F use.
# arg names the program; the suite checks the runner's terminal state for each. python@3.13's
# exec filter has no clock_nanosleep and admits clone only for threads: the idle program waits
# on a lock (futex), and fork dies of SIGSYS.
import os
import sys
import time


class Solution:
    def probe(self, arg: str) -> str:
        name, _, param = arg.partition(":")
        if name == "noop":
            return ""
        if name == "spin":
            ms = int(param or "0")
            deadline = time.perf_counter() + ms / 1000.0
            x = 0
            while ms == 0 or time.perf_counter() < deadline:
                for i in range(10000):
                    x += i ^ x
                x &= 0xFFFF
            return str(x & 1)
        if name == "sleep":
            import _thread

            lock = _thread.allocate_lock()
            lock.acquire()
            lock.acquire()
            return "woke"
        if name == "balloon":
            keep = []
            for _ in range(64):
                b = bytearray(16 << 20)
                b[::4096] = b"\x01" * ((len(b) + 4095) // 4096)
                keep.append(b)
            return "survived %d" % len(keep)
        if name == "tmpfsfill":
            buf = b"\0" * (512 << 10)
            i = 0
            while True:
                try:
                    with open("/w/f%d" % i, "wb") as f:
                        f.write(buf)
                except OSError as e:
                    return "enospc: " + repr(e)
                i += 1
        if name == "inodefill":
            i = 0
            while True:
                try:
                    open("/w/i%d" % i, "wb").close()
                except OSError as e:
                    return "enospc after %d: %r" % (i, e)
                i += 1
        if name == "stdoutflood":
            chunk = "x" * (64 << 10)
            while True:
                sys.stdout.write(chunk)
                sys.stdout.flush()
        if name == "threadbomb":
            import _thread

            lock = _thread.allocate_lock()
            lock.acquire()
            for _ in range(5000):
                _thread.start_new_thread(lock.acquire, ())
            lock.acquire()
            return "survived"
        if name == "forkbomb":
            for _ in range(1000):
                if os.fork() == 0:
                    time.sleep(3600)
            os._exit(3)
        if name == "orphan":
            if os.fork() == 0:
                if os.fork() == 0:
                    os.setsid()
                    time.sleep(3600)
                os._exit(0)
            return "parent done"
        return "bad probe " + arg
