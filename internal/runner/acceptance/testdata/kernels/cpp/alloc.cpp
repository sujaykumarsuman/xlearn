// Calibration kernel "alloc" (t3 §7.3, alloc/GC-heavy): four rounds of building a linked list of
// n/4 heap nodes, walking it and freeing it.
struct Cell {
    long long val;
    Cell* next;
};

class Solution {
public:
    long long kernel(int n) {
        uint32_t x = 99;
        long long s = 0;
        for (int r = 0; r < 4; r++) {
            Cell* head = nullptr;
            for (int i = 0; i < n / 4; i++) {
                x = x * 1664525u + 1013904223u;
                head = new Cell{(long long)(x >> 8), head};
            }
            for (Cell* c = head; c; c = c->next) s = (s + c->val) % 1000000007LL;
            while (head) {
                Cell* next = head->next;
                delete head;
                head = next;
            }
        }
        return s;
    }
};
