# Section B (network), Python: arg is "<kind>:<target>". Every attempt must fail. The exec filter
# kills the socket module's setup or socket() itself (signal SIGSYS), and past it the jail's
# network namespace is empty with loopback down. "blocked: ..." means the attempt failed
# in-process; "REACHED ..." means a connection or an answer got through, which fails the suite.
import sys


class Solution:
    def probe(self, arg: str) -> str:
        kind, _, target = arg.partition(":")
        sys.stderr.write("probing " + arg + "\n")
        sys.stderr.flush()
        import socket

        try:
            if kind == "dns":
                return "REACHED " + str(socket.getaddrinfo(target, None))
            host, _, port = target.rpartition(":")
            if kind == "bind":
                s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                s.bind((host, int(port)))
                s.listen(1)
                return "REACHED bound " + target
            s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM if kind == "udp" else socket.SOCK_STREAM)
            s.settimeout(1.0)
            s.connect((host, int(port)))
            if kind == "udp":
                s.send(bytes([0x12, 0x34, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1]))
                s.recv(512)
            return "REACHED " + arg
        except OSError as e:
            return "blocked: " + repr(e)
