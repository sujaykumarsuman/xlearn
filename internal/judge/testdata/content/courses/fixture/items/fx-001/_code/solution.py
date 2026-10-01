class Solution:
    def canSplit(self, nums: list[int]) -> bool:
        total = sum(nums)
        left = 0
        for x in nums[:-1]:
            left += x
            if 2 * left == total:
                return True
        return False
