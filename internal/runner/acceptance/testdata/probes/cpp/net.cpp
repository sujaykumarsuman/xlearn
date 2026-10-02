// Section B (network), C++: arg is "<kind>:<target>". Every attempt must fail. The exec filter
// kills socket() (signal SIGSYS), and past it the jail's network namespace is empty with loopback
// down. "blocked: ..." means the attempt failed in-process; "REACHED ..." means a connection or
// an answer got through, which fails the suite.
#include <arpa/inet.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>

class Solution {
    static string err(const string& what) { return "blocked: " + what + ": " + strerror(errno); }

    static bool addr(const string& target, sockaddr_in& sa) {
        auto colon = target.rfind(':');
        if (colon == string::npos) return false;
        memset(&sa, 0, sizeof sa);
        sa.sin_family = AF_INET;
        sa.sin_port = htons((uint16_t)stoi(target.substr(colon + 1)));
        return inet_pton(AF_INET, target.substr(0, colon).c_str(), &sa.sin_addr) == 1;
    }

public:
    string probe(string arg) {
        auto colon = arg.find(':');
        string kind = arg.substr(0, colon), target = arg.substr(colon + 1);
        fprintf(stderr, "probing %s\n", arg.c_str());
        if (kind == "tcp" || kind == "udp" || kind == "bind") {
            sockaddr_in sa;
            if (!addr(target, sa)) return "bad probe " + arg;
            int s = socket(AF_INET, kind == "udp" ? SOCK_DGRAM : SOCK_STREAM, 0);
            if (s < 0) return err("socket");
            if (kind == "bind") {
                if (bind(s, (sockaddr*)&sa, sizeof sa) < 0) return err("bind");
                if (listen(s, 1) < 0) return err("listen");
                return "REACHED bound " + target;
            }
            timeval tv{1, 0};
            setsockopt(s, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof tv);
            setsockopt(s, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof tv);
            if (connect(s, (sockaddr*)&sa, sizeof sa) < 0) return err("connect");
            if (kind == "udp") {
                const unsigned char q[] = {0x12, 0x34, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1};
                send(s, q, sizeof q, 0);
                char b[512];
                if (recv(s, b, sizeof b, 0) < 0) return err("no answer");
            }
            return "REACHED " + arg;
        }
        if (kind == "dns") {
            addrinfo* res = nullptr;
            int rc = getaddrinfo(target.c_str(), nullptr, nullptr, &res);
            if (rc != 0) return string("blocked: ") + gai_strerror(rc);
            freeaddrinfo(res);
            return "REACHED " + target;
        }
        return "bad probe " + arg;
    }
};
