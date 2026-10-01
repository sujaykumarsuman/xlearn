# want:
from __future__ import annotations

import collections.abc
import heapq as hq, bisect
from typing import List, Optional
from collections import (
    deque,
    defaultdict,
)
import sys; import math

s = "import os"
t = '''
import subprocess
'''
u = rb"from os import system"
# import socket


class Solution:
    def pairSum(self, nums: List[int], target: int) -> List[int]:
        def gen():
            yield from nums

        try:
            pass
        except ValueError as e:
            raise KeyError("x") from e
        return list(gen())[:0]
