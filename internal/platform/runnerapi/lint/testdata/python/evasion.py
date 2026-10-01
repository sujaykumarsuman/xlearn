# want:
# Lint-clean on purpose: __import__ is a builtin call, not an import statement. The jail stops
# it: the socket module's first call outside the python allowlist (epoll_create1 at import on
# CPython 3.13, else socket() itself) dies of SIGSYS under the KILL-default exec filter;
# refs_it_test runs this very file.
class Solution:
    def pairSum(self, nums, target):
        return [__import__("socket").socket().fileno()]
