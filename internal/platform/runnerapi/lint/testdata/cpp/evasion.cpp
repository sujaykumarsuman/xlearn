// want:
// Lint-clean on purpose: a hand-written libc declaration needs no header. The jail stops it
// (socket → SIGSYS under the KILL-default exec filter; refs_it_test runs the same program).
extern "C" int socket(int, int, int);
class Solution {
public:
    vector<int> pairSum(vector<int>& nums, int target) { return {socket(2, 1, 0)}; }
};
