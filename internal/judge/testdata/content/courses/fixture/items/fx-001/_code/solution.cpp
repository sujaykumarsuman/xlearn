class Solution {
public:
    bool canSplit(vector<int>& nums) {
        long long total = 0;
        for (int x : nums) total += x;
        long long left = 0;
        for (size_t i = 0; i + 1 < nums.size(); ++i) {
            left += nums[i];
            if (2 * left == total) return true;
        }
        return false;
    }
};
