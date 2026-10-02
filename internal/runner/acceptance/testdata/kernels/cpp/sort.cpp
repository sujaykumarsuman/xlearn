// Calibration kernel "sort" (t3 §7.3): sort n 32-bit LCG values, then a position-weighted sum.
class Solution {
public:
    long long kernel(int n) {
        vector<long long> a(n);
        uint32_t x = 12345;
        for (auto& v : a) {
            x = x * 1664525u + 1013904223u;
            v = x;
        }
        sort(a.begin(), a.end());
        long long s = 0;
        for (int i = 0; i < n; i++) s = (s + a[i] % 1000000007LL * (i + 1)) % 1000000007LL;
        return s;
    }
};
