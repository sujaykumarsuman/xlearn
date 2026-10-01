// want: TLE
// Every pair: O(n^2) on the perf cases.
class Solution {
public:
    vector<int> pairSum(vector<int>& nums, int target) {
        for (size_t i = 0; i < nums.size(); i++)
            for (size_t j = i + 1; j < nums.size(); j++)
                if ((long long)nums[i] + nums[j] == target) return {(int)i, (int)j};
        return {};
    }
};
