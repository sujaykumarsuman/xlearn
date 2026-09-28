A hash map trades space for time: it turns an O(n) membership scan into an O(1) average lookup. Most 'have I seen this?' and 'how many of each?' questions reduce to one pass that builds a `map[T]int` (a frequency map) or a `map[T]struct{}` (a set), then a second glance at the map.

The three moves you reuse constantly:
- **Seen-set**: add as you go, check before adding (duplicate detection).
- **Frequency map**: count occurrences, then reason over the counts.
- **Complement lookup**: for each element, ask whether the value you still need is already in the map (Two Sum).