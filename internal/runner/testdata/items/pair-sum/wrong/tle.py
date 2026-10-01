# want: TLE
# Every pair: O(n^2) on the perf cases.
class Solution:
    def pairSum(self, nums, target):
        n = len(nums)
        for i in range(n):
            for j in range(i + 1, n):
                if nums[i] + nums[j] == target:
                    return [i, j]
        return []
