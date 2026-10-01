// want: RE
// A failed assert aborts: RE by a signal, never a SIGSYS that would rotate the runner, so the cpp
// exec list carries abort's calls (decisions log, m3-04). The case process is its pid
// namespace's init, which ignores its own SIGABRT, so abort ends in its trap instruction
// (SIGTRAP on arm64, SIGSEGV on amd64).
class Solution {
public:
    vector<int> pairSum(vector<int>& nums, int target) {
        assert(nums.empty());
        return {};
    }
};
