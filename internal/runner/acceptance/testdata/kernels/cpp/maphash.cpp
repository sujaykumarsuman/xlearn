// Calibration kernel "maphash" (t3 §7.3, map-heavy): n updates then n lookups on a hash map with
// keys drawn from 4n values.
class Solution {
public:
    long long kernel(int n) {
        unordered_map<long long, long long> m;
        uint32_t x = 7;
        long long space = 4LL * n;
        for (int i = 0; i < n; i++) {
            x = x * 1664525u + 1013904223u;
            m[(long long)x % space] += i;
        }
        long long s = 0, hits = 0;
        for (int i = 0; i < n; i++) {
            x = x * 1664525u + 1013904223u;
            auto it = m.find((long long)x % space);
            if (it != m.end()) {
                hits++;
                s = (s + it->second) % 1000000007LL;
            }
        }
        return (s + hits) % 1000000007LL;
    }
};
