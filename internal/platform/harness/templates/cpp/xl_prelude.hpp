// xl_prelude.hpp: the C++ harness runtime of xlearn internal/platform/harness (func-json@1,
// class-ops@1). Public and versioned with the harness; DO NOT EDIT a copy. The generated
// zz_xl_harness.cpp includes it first, then defines the node types its signature uses, then
// includes the learner's solution.cpp, then main. It is one static file for every signature
// (a precompiled header can serve it, m3-15). Identifiers starting with xl are reserved.
//
// Wire format (README.md): one u32 big-endian framed canonical JSON frame in on fd 3, one out
// on fd 4: {"ok":<value>}, {"panic":"<class>"} or {"error":"<code>"}. The bytes match the Go
// harness exactly (the cross-language golden test is the guard).
#ifndef XL_PRELUDE_HPP
#define XL_PRELUDE_HPP

#include <bits/stdc++.h>
#include <unistd.h>

using namespace std;

namespace xlh {

constexpr std::size_t kMaxNodes = 1000000;
constexpr std::uint32_t kMaxFrame = 64u << 20;

// Fail is a harness error: {"error":"<code>"}.
struct Fail {
    const char* code;
};

[[noreturn]] inline void fail(const char* code) { throw Fail{code}; }

// ---- fd 3 / fd 4 ----

inline bool read_full(int fd, char* p, std::size_t n) {
    while (n > 0) {
        ssize_t k = ::read(fd, p, n);
        if (k < 0 && errno == EINTR) continue;
        if (k <= 0) return false;
        p += k;
        n -= static_cast<std::size_t>(k);
    }
    return true;
}

inline std::string read_input() {
    unsigned char h[4];
    if (!read_full(3, reinterpret_cast<char*>(h), 4)) fail("bad_input");
    std::uint32_t n = std::uint32_t(h[0]) << 24 | std::uint32_t(h[1]) << 16 | std::uint32_t(h[2]) << 8 | std::uint32_t(h[3]);
    if (n > kMaxFrame) fail("bad_input");
    std::string b(n, '\0');
    if (n > 0 && !read_full(3, b.data(), n)) fail("bad_input");
    return b;
}

inline void write_frame(const std::string& payload) {
    std::size_t n = payload.size();
    std::string out;
    out.reserve(n + 4);
    out += char((n >> 24) & 0xff);
    out += char((n >> 16) & 0xff);
    out += char((n >> 8) & 0xff);
    out += char(n & 0xff);
    out += payload;
    const char* p = out.data();
    std::size_t left = out.size();
    while (left > 0) {
        ssize_t k = ::write(4, p, left);
        if (k < 0 && errno == EINTR) continue;
        if (k <= 0) ::_exit(98);
        p += k;
        left -= static_cast<std::size_t>(k);
    }
}

// ---- UTF-8 (Go's unicode/utf8 rules) ----

inline void append_rune(std::string& out, std::uint32_t r) {
    if (r > 0x10ffff || (r >= 0xd800 && r <= 0xdfff)) r = 0xfffd;
    if (r < 0x80) {
        out += char(r);
    } else if (r < 0x800) {
        out += char(0xc0 | (r >> 6));
        out += char(0x80 | (r & 0x3f));
    } else if (r < 0x10000) {
        out += char(0xe0 | (r >> 12));
        out += char(0x80 | ((r >> 6) & 0x3f));
        out += char(0x80 | (r & 0x3f));
    } else {
        out += char(0xf0 | (r >> 18));
        out += char(0x80 | ((r >> 12) & 0x3f));
        out += char(0x80 | ((r >> 6) & 0x3f));
        out += char(0x80 | (r & 0x3f));
    }
}

inline bool valid_utf8(const std::string& s) {
    const unsigned char* p = reinterpret_cast<const unsigned char*>(s.data());
    std::size_t n = s.size(), i = 0;
    while (i < n) {
        unsigned char c = p[i];
        if (c < 0x80) {
            i++;
            continue;
        }
        std::size_t len;
        unsigned char lo = 0x80, hi = 0xbf;
        if (c >= 0xc2 && c <= 0xdf) {
            len = 2;
        } else if (c >= 0xe0 && c <= 0xef) {
            len = 3;
            if (c == 0xe0) lo = 0xa0;
            if (c == 0xed) hi = 0x9f;
        } else if (c >= 0xf0 && c <= 0xf4) {
            len = 4;
            if (c == 0xf0) lo = 0x90;
            if (c == 0xf4) hi = 0x8f;
        } else {
            return false;
        }
        if (i + len > n) return false;
        if (p[i + 1] < lo || p[i + 1] > hi) return false;
        for (std::size_t k = 2; k < len; k++) {
            if (p[i + k] < 0x80 || p[i + k] > 0xbf) return false;
        }
        i += len;
    }
    return true;
}

// ---- the reader (canonical JSON in; strict where the Go harness is) ----

struct P {
    std::string b;
    std::size_t i = 0;

    explicit P(std::string s) : b(std::move(s)) {}

    void ws() {
        while (i < b.size() && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r')) i++;
    }
    char peek() {
        ws();
        if (i >= b.size()) fail("bad_input");
        return b[i];
    }
    void expect(char c) {
        if (peek() != c) fail("bad_input");
        i++;
    }
    void end() {
        ws();
        if (i != b.size()) fail("bad_input");
    }
    bool lit(const char* s) {
        ws();
        std::size_t n = std::strlen(s);
        if (b.size() - i >= n && b.compare(i, n, s) == 0) {
            i += n;
            return true;
        }
        return false;
    }
    bool null() { return lit("null"); }

    std::string num_text() {
        ws();
        std::size_t s = i;
        while (i < b.size()) {
            char c = b[i];
            if ((c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E') {
                i++;
                continue;
            }
            break;
        }
        if (s == i) fail("bad_input");
        return b.substr(s, i - s);
    }

    long long i64(long long lo, long long hi) {
        std::string t = num_text();
        std::size_t k = (t[0] == '-' || t[0] == '+') ? 1 : 0;
        if (k == t.size()) fail("bad_input");
        for (std::size_t j = k; j < t.size(); j++) {
            if (t[j] < '0' || t[j] > '9') fail("bad_input");
        }
        errno = 0;
        char* endp = nullptr;
        long long v = std::strtoll(t.c_str(), &endp, 10);
        if (errno != 0 || endp != t.c_str() + t.size() || v < lo || v > hi) fail("bad_input");
        return v;
    }

    double f64() {
        std::string t = num_text();
        errno = 0;
        char* endp = nullptr;
        double v = std::strtod(t.c_str(), &endp);
        if (endp != t.c_str() + t.size() || !std::isfinite(v)) fail("bad_input");
        return v;
    }

    std::uint32_t hex4() {
        if (i + 4 > b.size()) fail("bad_input");
        std::uint32_t r = 0;
        for (int k = 0; k < 4; k++) {
            char c = b[i + k];
            r <<= 4;
            if (c >= '0' && c <= '9') r |= std::uint32_t(c - '0');
            else if (c >= 'a' && c <= 'f') r |= std::uint32_t(c - 'a' + 10);
            else if (c >= 'A' && c <= 'F') r |= std::uint32_t(c - 'A' + 10);
            else fail("bad_input");
        }
        i += 4;
        return r;
    }

    std::string str() {
        expect('"');
        std::string out;
        for (;;) {
            if (i >= b.size()) fail("bad_input");
            unsigned char c = static_cast<unsigned char>(b[i++]);
            if (c == '"') {
                if (!valid_utf8(out)) fail("bad_input");
                return out;
            }
            if (c == '\\') {
                if (i >= b.size()) fail("bad_input");
                char e = b[i++];
                switch (e) {
                case '"': case '\\': case '/': out += e; break;
                case 'b': out += '\b'; break;
                case 'f': out += '\f'; break;
                case 'n': out += '\n'; break;
                case 'r': out += '\r'; break;
                case 't': out += '\t'; break;
                case 'u': {
                    std::uint32_t r = hex4();
                    if (r >= 0xd800 && r < 0xdc00) {
                        if (i + 6 > b.size() || b[i] != '\\' || b[i + 1] != 'u') fail("bad_input");
                        i += 2;
                        std::uint32_t r2 = hex4();
                        if (r2 < 0xdc00 || r2 > 0xdfff) fail("bad_input");
                        r = ((r - 0xd800) << 10 | (r2 - 0xdc00)) + 0x10000;
                    }
                    append_rune(out, r);
                    break;
                }
                default: fail("bad_input");
                }
                continue;
            }
            if (c < 0x20) fail("bad_input");
            out += char(c);
        }
    }

    // skip passes over one JSON value.
    void skip(int depth) {
        if (depth > 64) fail("bad_input");
        char c = peek();
        if (c == '"') {
            str();
        } else if (c == '[' || c == '{') {
            i++;
            char close = c == '{' ? '}' : ']';
            if (peek() == close) {
                i++;
                return;
            }
            for (;;) {
                if (c == '{') {
                    str();
                    expect(':');
                }
                skip(depth + 1);
                if (peek() == ',') {
                    i++;
                    continue;
                }
                expect(close);
                return;
            }
        } else if (c == 't') {
            if (!lit("true")) fail("bad_input");
        } else if (c == 'f') {
            if (!lit("false")) fail("bad_input");
        } else if (c == 'n') {
            if (!null()) fail("bad_input");
        } else {
            num_text();
        }
    }

    // spans splits an array into its elements' raw bytes.
    std::vector<std::string> spans() {
        expect('[');
        std::vector<std::string> out;
        if (peek() == ']') {
            i++;
            return out;
        }
        for (;;) {
            ws();
            std::size_t s = i;
            skip(1);
            out.push_back(b.substr(s, i - s));
            if (peek() == ',') {
                i++;
                continue;
            }
            expect(']');
            return out;
        }
    }
};

// ---- the writer (canonical JSON out) ----

struct W {
    std::string b;
    std::size_t nodes = 0;

    void raw(const char* s) { b += s; }
    void node() {
        if (++nodes > kMaxNodes) fail("too_many_nodes");
    }
};

// fmt_float is Go's strconv.FormatFloat(v, 'g', -1, 64) for a finite non-zero v: the
// shortest round-trip digits, %e when the exponent is < -4 or >= 6 (two exponent digits at
// least), %f otherwise.
inline void fmt_float(std::string& out, double v) {
    char buf[64];
    auto r = std::to_chars(buf, buf + sizeof buf, v, std::chars_format::scientific);
    std::string s(buf, r.ptr);
    bool neg = s[0] == '-';
    if (neg) s.erase(0, 1);
    std::size_t e = s.find('e');
    std::string digs;
    for (std::size_t k = 0; k < e; k++) {
        if (s[k] != '.') digs += s[k];
    }
    int ex = std::stoi(s.substr(e + 1));
    while (digs.size() > 1 && digs.back() == '0') digs.pop_back();
    int nd = int(digs.size());
    int dp = ex + 1;
    if (neg) out += '-';
    int exp = dp - 1;
    if (exp < -4 || exp >= 6) {
        out += digs[0];
        if (nd > 1) {
            out += '.';
            out.append(digs, 1, std::string::npos);
        }
        out += 'e';
        out += exp < 0 ? '-' : '+';
        int a = exp < 0 ? -exp : exp;
        if (a < 10) {
            out += '0';
            out += char('0' + a);
        } else {
            out += std::to_string(a);
        }
        return;
    }
    int prec = std::max(nd - dp, 0);
    if (dp > 0) {
        for (int k = 0; k < dp; k++) out += k < nd ? digs[k] : '0';
    } else {
        out += '0';
    }
    if (prec > 0) {
        out += '.';
        for (int k = 0; k < prec; k++) {
            int j = dp + k;
            out += (j < 0 || j >= nd) ? '0' : digs[j];
        }
    }
}

// ---- codecs: C<T>::dec(P&) and C<T>::enc(W&, const T&) ----

template <class T>
struct C;

template <>
struct C<int> {
    static int dec(P& p) { return int(p.i64(INT32_MIN, INT32_MAX)); }
    static void enc(W& w, int v) { w.b += std::to_string(v); }
};

template <>
struct C<long long> {
    static long long dec(P& p) { return p.i64(LLONG_MIN, LLONG_MAX); }
    static void enc(W& w, long long v) { w.b += std::to_string(v); }
};

template <>
struct C<double> {
    static double dec(P& p) { return p.f64(); }
    static void enc(W& w, double v) {
        if (!std::isfinite(v)) fail("non_finite");
        if (v == 0) {
            w.b += '0';
            return;
        }
        fmt_float(w.b, v);
    }
};

template <>
struct C<bool> {
    static bool dec(P& p) {
        if (p.lit("true")) return true;
        if (p.lit("false")) return false;
        fail("bad_input");
    }
    static void enc(W& w, bool v) { w.b += v ? "true" : "false"; }
};

template <>
struct C<std::string> {
    static std::string dec(P& p) { return p.str(); }
    static void enc(W& w, const std::string& v) {
        if (!valid_utf8(v)) fail("invalid_utf8");
        static const char hex[] = "0123456789abcdef";
        w.b += '"';
        for (unsigned char c : v) {
            if (c == '"' || c == '\\') {
                w.b += '\\';
                w.b += char(c);
            } else if (c < 0x20) {
                w.b += "\\u00";
                w.b += hex[c >> 4];
                w.b += hex[c & 0xf];
            } else {
                w.b += char(c);
            }
        }
        w.b += '"';
    }
};

template <class T>
struct C<std::vector<T>> {
    static std::vector<T> dec(P& p) {
        p.expect('[');
        std::vector<T> out;
        if (p.peek() == ']') {
            p.i++;
            return out;
        }
        for (;;) {
            out.push_back(C<T>::dec(p));
            if (p.peek() == ',') {
                p.i++;
                continue;
            }
            p.expect(']');
            return out;
        }
    }
    static void enc(W& w, const std::vector<T>& v) {
        w.b += '[';
        bool first = true;
        for (const auto& e : v) {
            if (!first) w.b += ',';
            first = false;
            C<T>::enc(w, e);
        }
        w.b += ']';
    }
};

// Node codecs, instantiated by the generated harness for the node types its signature uses
// (ListNode{val, next}, TreeNode{val, left, right}, Node{val, neighbors}).

template <class N>
struct ListCodec {
    static N* dec(P& p) {
        std::vector<int> vals = C<std::vector<int>>::dec(p);
        N* head = nullptr;
        for (std::size_t k = vals.size(); k-- > 0;) {
            N* n = new N(vals[k]);
            n->next = head;
            head = n;
        }
        return head;
    }
    static void enc(W& w, N* head) {
        std::unordered_set<const N*> seen;
        w.b += '[';
        for (const N* n = head; n != nullptr; n = n->next) {
            if (!seen.insert(n).second) fail("cycle");
            w.node();
            if (n != head) w.b += ',';
            w.b += std::to_string(static_cast<long long>(n->val));
        }
        w.b += ']';
    }
};

template <class N>
struct TreeCodec {
    static N* dec(P& p) {
        p.expect('[');
        std::vector<std::optional<int>> vals;
        if (p.peek() == ']') {
            p.i++;
        } else {
            for (;;) {
                if (p.null()) vals.emplace_back();
                else vals.emplace_back(C<int>::dec(p));
                if (p.peek() == ',') {
                    p.i++;
                    continue;
                }
                p.expect(']');
                break;
            }
        }
        if (vals.empty() || !vals[0]) return nullptr;
        N* root = new N(*vals[0]);
        std::deque<N*> q{root};
        for (std::size_t k = 1; !q.empty() && k < vals.size();) {
            N* n = q.front();
            q.pop_front();
            if (vals[k]) {
                n->left = new N(*vals[k]);
                q.push_back(n->left);
            }
            k++;
            if (k < vals.size() && vals[k]) {
                n->right = new N(*vals[k]);
                q.push_back(n->right);
            }
            k++;
        }
        return root;
    }
    static void enc(W& w, N* root) {
        if (root == nullptr) {
            w.b += "[]";
            return;
        }
        std::unordered_set<const N*> seen;
        std::vector<const N*> q{root};
        for (std::size_t k = 0; k < q.size(); k++) {
            const N* n = q[k];
            if (n == nullptr) continue;
            if (!seen.insert(n).second) fail("cycle");
            w.node();
            q.push_back(n->left);
            q.push_back(n->right);
        }
        std::size_t end = q.size();
        while (end > 0 && q[end - 1] == nullptr) end--;
        w.b += '[';
        for (std::size_t k = 0; k < end; k++) {
            if (k > 0) w.b += ',';
            if (q[k] == nullptr) w.b += "null";
            else w.b += std::to_string(static_cast<long long>(q[k]->val));
        }
        w.b += ']';
    }
};

template <class N>
struct GraphCodec {
    static N* dec(P& p) {
        std::vector<std::vector<int>> adj = C<std::vector<std::vector<int>>>::dec(p);
        if (adj.empty()) return nullptr;
        std::vector<N*> nodes(adj.size());
        for (std::size_t k = 0; k < nodes.size(); k++) nodes[k] = new N(int(k + 1));
        for (std::size_t k = 0; k < adj.size(); k++) {
            for (int m : adj[k]) {
                if (m < 1 || std::size_t(m) > adj.size()) fail("bad_input");
                nodes[k]->neighbors.push_back(nodes[std::size_t(m) - 1]);
            }
        }
        return nodes[0];
    }
    static void enc(W& w, N* start) {
        if (start == nullptr) {
            w.b += "[]";
            return;
        }
        std::unordered_map<long long, const N*> by_val;
        std::unordered_set<const N*> seen{start};
        std::vector<const N*> q{start};
        for (std::size_t k = 0; k < q.size(); k++) {
            const N* n = q[k];
            w.node();
            long long v = n->val;
            if (by_val.count(v)) fail("invalid_graph");
            by_val[v] = n;
            for (const N* m : n->neighbors) {
                if (m == nullptr) fail("invalid_graph");
                if (seen.insert(m).second) q.push_back(m);
            }
        }
        w.b += '[';
        for (std::size_t v = 1; v <= q.size(); v++) {
            auto it = by_val.find(static_cast<long long>(v));
            if (it == by_val.end()) fail("invalid_graph");
            if (v > 1) w.b += ',';
            w.b += '[';
            bool first = true;
            for (const N* m : it->second->neighbors) {
                if (!first) w.b += ',';
                first = false;
                w.b += std::to_string(static_cast<long long>(m->val));
            }
            w.b += ']';
        }
        w.b += ']';
    }
};

// ---- the case driver ----

inline std::string panic_frame(const char* cls) { return std::string("{\"panic\":\"") + cls + "\"}"; }

// run calls body with a writer and writes exactly one frame on fd 4: the body's {"ok"} frame,
// a harness {"error"} or a caught exception's {"panic"} class. Uncatchable failures (a signal,
// a stack overflow, std::terminate) leave fd 4 empty.
template <class F>
int run(F body) {
    std::string out;
    try {
        W w;
        body(w);
        out.swap(w.b);
    } catch (const Fail& f) {
        out = std::string("{\"error\":\"") + f.code + "\"}";
    } catch (const std::out_of_range&) {
        out = panic_frame("out_of_range");
    } catch (const std::length_error&) {
        out = panic_frame("length_error");
    } catch (const std::logic_error&) {
        out = panic_frame("logic_error");
    } catch (const std::bad_alloc&) {
        out = panic_frame("bad_alloc");
    } catch (const std::runtime_error&) {
        out = panic_frame("runtime_error");
    } catch (...) {
        out = panic_frame("other");
    }
    write_frame(out);
    return 0;
}

}  // namespace xlh

#endif  // XL_PRELUDE_HPP
