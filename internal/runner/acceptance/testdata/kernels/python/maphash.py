# Calibration kernel "maphash" (t3 §7.3, map-heavy): n updates then n lookups on a hash map with
# keys drawn from 4n values.
class Solution:
    def kernel(self, n: int) -> int:
        m = {}
        x = 7
        space = 4 * n
        for i in range(n):
            x = (x * 1664525 + 1013904223) & 0xFFFFFFFF
            k = x % space
            m[k] = m.get(k, 0) + i
        s = hits = 0
        for _ in range(n):
            x = (x * 1664525 + 1013904223) & 0xFFFFFFFF
            v = m.get(x % space)
            if v is not None:
                hits += 1
                s = (s + v) % 1000000007
        return (s + hits) % 1000000007
