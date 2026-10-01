// want:
#include <bits/stdc++.h>
#include <vector>
#include <cstdio>
#include <math.h>
#pragma GCC optimize("O3")
#pragma GCC target("avx2")
// A comment that says asm, main, #include "x.h" and _Pragma is fine.
/* so is a block: asm volatile("syscall"); int main() {} */
#define SQ(x) ((x) * (x))
static const char* s = "asm main #include \"a.h\"";
static const char* r = R"xl(asm(") main)xl";
static const char c = '"';
long long remain = 1'000'000;
class Solution {
public:
    vector<int> pairSum(vector<int>& nums, int target) { int domain = SQ(target); (void)domain; return {}; }
};
