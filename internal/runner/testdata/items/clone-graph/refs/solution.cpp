class Solution {
public:
    Node* cloneGraph(Node* node) {
        if (!node) return nullptr;
        unordered_map<Node*, Node*> copies{{node, new Node(node->val)}};
        deque<Node*> queue{node};
        while (!queue.empty()) {
            Node* n = queue.front();
            queue.pop_front();
            for (Node* m : n->neighbors) {
                auto it = copies.find(m);
                if (it == copies.end()) {
                    it = copies.emplace(m, new Node(m->val)).first;
                    queue.push_back(m);
                }
                copies[n]->neighbors.push_back(it->second);
            }
        }
        return copies[node];
    }
};
