# Calibration kernel "bfs" (t3 §7.3): BFS from vertex 0 over n random undirected edges on n/4
# vertices; the sum of (distance + 1) over the reached vertices.
class Solution:
    def kernel(self, n: int) -> int:
        v = n // 4
        adj = [[] for _ in range(v)]
        x = 2024
        for _ in range(n):
            x = (x * 1664525 + 1013904223) & 0xFFFFFFFF
            a = x % v
            x = (x * 1664525 + 1013904223) & 0xFFFFFFFF
            b = x % v
            adj[a].append(b)
            adj[b].append(a)
        dist = [-1] * v
        dist[0] = 0
        queue = [0]
        s = 0
        h = 0
        while h < len(queue):
            u = queue[h]
            h += 1
            s += dist[u] + 1
            du = dist[u] + 1
            for w in adj[u]:
                if dist[w] < 0:
                    dist[w] = du
                    queue.append(w)
        return s % 1000000007
