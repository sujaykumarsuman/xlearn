class Solution {
public:
    vector<int> pairSum(vector<int>& nums, int target) {
        unordered_map<long long, int> seen;
        seen.reserve(nums.size() * 2);
        for (int i = 0; i < (int)nums.size(); i++) {
            auto it = seen.find((long long)target - nums[i]);
            if (it != seen.end()) return {it->second, i};
            seen.emplace(nums[i], i);
        }
        return {};
    }
};
