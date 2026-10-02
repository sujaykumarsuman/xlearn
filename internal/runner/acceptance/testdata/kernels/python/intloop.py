# Calibration kernel "intloop" (t3 §7.3): n steps of a 32-bit LCG, summing the high halves.
class Solution:
    def kernel(self, n: int) -> int:
        x = 1
        s = 0
        for _ in range(n):
            x = (x * 1664525 + 1013904223) & 0xFFFFFFFF
            s += x >> 16
        return s % 1000000007
