from collections import deque
from typing import Optional


class Solution:
    def cloneGraph(self, node: Optional["Node"]) -> Optional["Node"]:
        if node is None:
            return None
        copies = {node: Node(node.val)}
        queue = deque([node])
        while queue:
            n = queue.popleft()
            for m in n.neighbors:
                c = copies.get(m)
                if c is None:
                    c = copies[m] = Node(m.val)
                    queue.append(m)
                copies[n].neighbors.append(c)
        return copies[node]
