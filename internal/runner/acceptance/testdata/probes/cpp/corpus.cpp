// Section D (t3 §9's P2 corpus), C++, plus the no-op and spin programs sections A and F use. arg
// names the program; the suite checks the runner's terminal state for each. cpp@g++14's exec
// filter has no openat, clone or clock_nanosleep: the file fills, the bombs and the orphan die
// of SIGSYS, and the idle program waits on a futex.
#include <linux/futex.h>
#include <sys/syscall.h>

class Solution {
public:
    string probe(string arg) {
        auto colon = arg.find(':');
        string name = arg.substr(0, colon), param = colon == string::npos ? "" : arg.substr(colon + 1);
        if (name == "noop") return "";
        if (name == "spin") {
            long ms = param.empty() ? 0 : stol(param);
            auto deadline = chrono::steady_clock::now() + chrono::milliseconds(ms);
            volatile unsigned long x = 0;
            while (ms == 0 || chrono::steady_clock::now() < deadline)
                for (int i = 0; i < 100000; i++) x = x + (i ^ x);
            return to_string(x & 1);
        }
        if (name == "sleep") {
            static int word = 0;
            for (;;) syscall(SYS_futex, &word, FUTEX_WAIT_PRIVATE, 0, nullptr, nullptr, 0);
        }
        if (name == "balloon") {
            vector<char*> keep;
            for (long n = 0; n < (1L << 30); n += 16 << 20) {
                char* b = new char[16 << 20];
                for (long i = 0; i < (16 << 20); i += 4096) b[i] = 1;
                keep.push_back(b);
            }
            return "survived " + to_string(keep.size());
        }
        if (name == "tmpfsfill") {
            static char buf[512 << 10];
            for (int i = 0;; i++) {
                FILE* f = fopen(("/w/f" + to_string(i)).c_str(), "wb");
                if (!f) return string("enospc: ") + strerror(errno);
                size_t w = fwrite(buf, 1, sizeof buf, f);
                if (fclose(f) != 0 || w != sizeof buf) return string("enospc: ") + strerror(errno);
            }
        }
        if (name == "inodefill") {
            for (int i = 0;; i++) {
                FILE* f = fopen(("/w/i" + to_string(i)).c_str(), "wb");
                if (!f) return "enospc after " + to_string(i) + ": " + strerror(errno);
                fclose(f);
            }
        }
        if (name == "stdoutflood") {
            static char buf[64 << 10];
            memset(buf, 'x', sizeof buf);
            for (;;)
                if (fwrite(buf, 1, sizeof buf, stdout) != sizeof buf) return "write failed";
        }
        if (name == "threadbomb") {
            vector<thread> ts;
            for (int i = 0; i < 5000; i++) ts.emplace_back([] { for (;;) pause(); });
            for (auto& t : ts) t.join();
            return "survived";
        }
        if (name == "forkbomb") {
            for (int i = 0; i < 1000; i++)
                if (fork() == 0) for (;;) pause();
            _exit(3);
        }
        if (name == "orphan") {
            pid_t p = fork();
            if (p == 0) {
                if (fork() == 0) { setsid(); for (;;) pause(); }
                _exit(0);
            }
            return "parent done";
        }
        return "bad probe " + arg;
    }
};
