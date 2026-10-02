# Calibration kernel "sort" (t3 §7.3): sort n 32-bit LCG values, then a position-weighted sum.
class Solution:
    def kernel(self, n: int) -> int:
        a = [0] * n
        x = 12345
        for i in range(n):
            x = (x * 1664525 + 1013904223) & 0xFFFFFFFF
            a[i] = x
        a.sort()
        s = 0
        for i, v in enumerate(a):
            s = (s + v % 1000000007 * (i + 1)) % 1000000007
        return s
