// Calibration kernel "bfs" (t3 §7.3): BFS from vertex 0 over n random undirected edges on n/4
// vertices; the sum of (distance + 1) over the reached vertices.
class Solution {
public:
    long long kernel(int n) {
        int v = n / 4;
        vector<vector<int>> adj(v);
        uint32_t x = 2024;
        for (int i = 0; i < n; i++) {
            x = x * 1664525u + 1013904223u;
            int a = x % (uint32_t)v;
            x = x * 1664525u + 1013904223u;
            int b = x % (uint32_t)v;
            adj[a].push_back(b);
            adj[b].push_back(a);
        }
        vector<int> dist(v, -1);
        dist[0] = 0;
        vector<int> queue{0};
        long long s = 0;
        for (size_t h = 0; h < queue.size(); h++) {
            int u = queue[h];
            s += dist[u] + 1;
            for (int w : adj[u])
                if (dist[w] < 0) {
                    dist[w] = dist[u] + 1;
                    queue.push_back(w);
                }
        }
        return s % 1000000007LL;
    }
};
