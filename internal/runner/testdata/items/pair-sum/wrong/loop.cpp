// want: TLE
// The volatile counter keeps the loop (C++ may assume a side-effect-free loop terminates).
class Solution {
public:
    vector<int> pairSum(vector<int>& nums, int target) {
        volatile unsigned long long spins = 0;
        for (;;) spins = spins + 1;
    }
};
