class Solution {
public:
    vector<vector<int>> levelOrder(TreeNode* root) {
        vector<vector<int>> out;
        if (!root) return out;
        vector<TreeNode*> level{root};
        while (!level.empty()) {
            vector<TreeNode*> next;
            vector<int> vals;
            vals.reserve(level.size());
            for (TreeNode* n : level) {
                vals.push_back(n->val);
                if (n->left) next.push_back(n->left);
                if (n->right) next.push_back(n->right);
            }
            out.push_back(move(vals));
            level.swap(next);
        }
        return out;
    }
};
