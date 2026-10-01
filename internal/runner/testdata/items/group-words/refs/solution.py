from typing import List


class Solution:
    def groupWords(self, words: List[str]) -> List[List[str]]:
        index = {}
        out = []
        for w in words:
            k = bytes(sorted(w.encode("utf-8")))
            i = index.get(k)
            if i is None:
                i = len(out)
                index[k] = i
                out.append([])
            out[i].append(w)
        return out
