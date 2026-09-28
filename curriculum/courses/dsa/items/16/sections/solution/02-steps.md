1) Sort nums.
2) Loop i from 0 to n-3; if nums[i] == nums[i-1], skip (duplicate anchor).
3) Early exit: once nums[i] > 0, no triple can sum to zero.
4) Two-pointer l, r inward on target -nums[i].
5) On a hit, append the triplet, then skip equal values at l and r before moving on.