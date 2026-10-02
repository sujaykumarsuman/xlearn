// Calibration kernel "intloop" (t3 §7.3): n steps of a 32-bit LCG, summing the high halves.
class Solution {
public:
    long long kernel(int n) {
        uint32_t x = 1;
        long long s = 0;
        for (int i = 0; i < n; i++) {
            x = x * 1664525u + 1013904223u;
            s += x >> 16;
        }
        return s % 1000000007LL;
    }
};
