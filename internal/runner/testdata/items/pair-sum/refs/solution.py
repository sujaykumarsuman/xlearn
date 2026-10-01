from typing import List


class Solution:
    def pairSum(self, nums: List[int], target: int) -> List[int]:
        seen = {}
        for i, x in enumerate(nums):
            j = seen.get(target - x)
            if j is not None:
                return [j, i]
            if x not in seen:
                seen[x] = i
        return []
