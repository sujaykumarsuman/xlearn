class Solution {
public:
    vector<double> runningMedian(vector<int>& nums) {
        priority_queue<long long> lo;
        priority_queue<long long, vector<long long>, greater<long long>> hi;
        vector<double> out;
        out.reserve(nums.size());
        for (int x : nums) {
            if (lo.empty() || x <= lo.top()) lo.push(x);
            else hi.push(x);
            if (lo.size() > hi.size() + 1) { hi.push(lo.top()); lo.pop(); }
            else if (hi.size() > lo.size()) { lo.push(hi.top()); hi.pop(); }
            if (lo.size() > hi.size()) out.push_back((double)lo.top());
            else out.push_back((double)(lo.top() + hi.top()) / 2);
        }
        return out;
    }
};
