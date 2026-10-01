// want: cpp_asm cpp_asm cpp_asm cpp_asm
#define NOP __asm__("nop")
class Solution {
public:
    int f() { asm("nop"); __asm__("nop"); __asm("nop"); NOP; return 0; }
};
