// want: RE
// A real null load (the volatile pointer keeps -O2 from turning it into a trap): SIGSEGV.
class Solution {
public:
    ListNode* reverseList(ListNode* head) {
        ListNode* volatile p = nullptr;
        return p->next;
    }
};
