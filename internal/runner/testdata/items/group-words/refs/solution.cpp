class Solution {
public:
    vector<vector<string>> groupWords(vector<string>& words) {
        unordered_map<string, int> index;
        vector<vector<string>> out;
        for (const string& w : words) {
            string k = w;
            sort(k.begin(), k.end(), [](char a, char b) { return (unsigned char)a < (unsigned char)b; });
            auto it = index.find(k);
            int i;
            if (it == index.end()) {
                i = out.size();
                index.emplace(k, i);
                out.emplace_back();
            } else {
                i = it->second;
            }
            out[i].push_back(w);
        }
        return out;
    }
};
