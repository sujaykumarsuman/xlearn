// Section E (cross-job markers, t3 §5.10), C++: arg is "<write|read>:<channel>:<token>". A write
// answers "written" or "denied: ..." (or dies of SIGSYS: cpp@g++14's exec filter has no openat,
// shmget, mq_open, socket or add_key); a read answers "visible" or "absent: ...". Job N's
// markers must be invisible to job N+1.
#include <fcntl.h>
#include <mqueue.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <sys/socket.h>
#include <sys/syscall.h>
#include <sys/un.h>

class Solution {
    static string e(const string& what) { return what + ": " + strerror(errno); }

    static unsigned fnv(const string& s) {
        unsigned h = 2166136261u;
        for (unsigned char c : s) { h ^= c; h *= 16777619u; }
        return h & 0x7fffffff;
    }

public:
    string probe(string arg) {
        auto c1 = arg.find(':'), c2 = arg.find(':', c1 + 1);
        if (c1 == string::npos || c2 == string::npos) return "bad probe " + arg;
        string op = arg.substr(0, c1), ch = arg.substr(c1 + 1, c2 - c1 - 1), tok = arg.substr(c2 + 1);
        string name = "xl-" + tok;
        bool write = op == "write";
        if (ch == "w" || ch == "tmp" || ch == "devshm") {
            string path = (ch == "w" ? "/w/" : ch == "tmp" ? "/tmp/" : "/dev/shm/") + name;
            if (write) {
                FILE* f = fopen(path.c_str(), "wb");
                if (!f) return e("denied");
                fputs(tok.c_str(), f);
                fclose(f);
                return "written";
            }
            FILE* f = fopen(path.c_str(), "rb");
            if (!f) return e("absent");
            fclose(f);
            return "visible";
        }
        if (ch == "sysvshm") {
            if (write) return shmget(fnv(tok), 4096, IPC_CREAT | 0666) < 0 ? e("denied") : "written";
            return shmget(fnv(tok), 0, 0) < 0 ? e("absent") : "visible";
        }
        if (ch == "posixmq") {
            string q = "/" + name;
            if (write) return mq_open(q.c_str(), O_CREAT | O_RDWR, 0666, nullptr) == (mqd_t)-1 ? e("denied") : "written";
            return mq_open(q.c_str(), O_RDONLY) == (mqd_t)-1 ? e("absent") : "visible";
        }
        if (ch == "abstract") {
            int s = socket(AF_UNIX, SOCK_STREAM, 0);
            if (s < 0) return e("denied");
            sockaddr_un sa{};
            sa.sun_family = AF_UNIX;
            memcpy(sa.sun_path + 1, name.data(), min(name.size(), sizeof sa.sun_path - 2));
            socklen_t len = offsetof(sockaddr_un, sun_path) + 1 + min(name.size(), sizeof sa.sun_path - 2);
            if (write) return bind(s, (sockaddr*)&sa, len) < 0 || listen(s, 1) < 0 ? e("denied") : "written";
            return connect(s, (sockaddr*)&sa, len) < 0 ? e("absent") : "visible";
        }
        if (ch == "keyring") {
            if (write)
                return syscall(SYS_add_key, "user", name.c_str(), tok.c_str(), tok.size(), -4) < 0 ? e("denied") : "written";
            return syscall(SYS_keyctl, 10, -4, "user", name.c_str(), 0) < 0 ? e("absent") : "visible";
        }
        return "bad probe " + arg;
    }
};
