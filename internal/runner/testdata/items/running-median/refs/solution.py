import heapq
from typing import List


class Solution:
    def runningMedian(self, nums: List[int]) -> List[float]:
        lo, hi = [], []  # lo holds negated values (a max-heap)
        out = []
        for x in nums:
            if not lo or x <= -lo[0]:
                heapq.heappush(lo, -x)
            else:
                heapq.heappush(hi, x)
            if len(lo) > len(hi) + 1:
                heapq.heappush(hi, -heapq.heappop(lo))
            elif len(hi) > len(lo):
                heapq.heappush(lo, -heapq.heappop(hi))
            if len(lo) > len(hi):
                out.append(float(-lo[0]))
            else:
                out.append((-lo[0] + hi[0]) / 2)
        return out
