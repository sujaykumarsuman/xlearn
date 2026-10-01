// want: cpp_directive cpp_directive cpp_directive cpp_directive cpp_directive cpp_directive
#include_next <vector>
#embed "solution.cpp"
#line 1 "other.cpp"
#pragma once
%:pragma GCC push_options
#import <vector>
class Solution {};
