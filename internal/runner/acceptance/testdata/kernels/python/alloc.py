# Calibration kernel "alloc" (t3 §7.3, alloc/GC-heavy): four rounds of building a linked list of
# n/4 heap nodes and walking it.
class Cell:
    __slots__ = ("val", "next")

    def __init__(self, val, nxt):
        self.val = val
        self.next = nxt


class Solution:
    def kernel(self, n: int) -> int:
        x = 99
        s = 0
        for _ in range(4):
            head = None
            for _ in range(n // 4):
                x = (x * 1664525 + 1013904223) & 0xFFFFFFFF
                head = Cell(x >> 8, head)
            c = head
            while c is not None:
                s = (s + c.val) % 1000000007
                c = c.next
            # Unlink iteratively: a 250k-deep chain would recurse in the deallocator.
            while head is not None:
                head.next, head = None, head.next
        return s
