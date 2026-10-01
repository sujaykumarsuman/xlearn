class Solution {
public:
    vector<vector<int>> trios(vector<int>& nums, int target) {
        vector<int> a(nums);
        sort(a.begin(), a.end());
        vector<vector<int>> out;
        for (size_t i = 0; i + 2 < a.size(); ++i) {
            if (i > 0 && a[i] == a[i - 1]) continue;
            size_t l = i + 1, r = a.size() - 1;
            while (l < r) {
                long long s = (long long)a[i] + a[l] + a[r];
                if (s < target) {
                    ++l;
                } else if (s > target) {
                    --r;
                } else {
                    out.push_back({a[i], a[l], a[r]});
                    while (l < r && a[l] == a[l + 1]) ++l;
                    while (l < r && a[r] == a[r - 1]) --r;
                    ++l;
                    --r;
                }
            }
        }
        return out;
    }
};
