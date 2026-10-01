from collections import deque


class FixedShelf:
    def __init__(self, capacity: int):
        self.capacity = capacity
        self.values = {}
        self.order = deque()

    def Put(self, key: int, value: int) -> None:
        if key in self.values:
            self.values[key] = value
            return
        if len(self.order) == self.capacity:
            del self.values[self.order.popleft()]
        self.order.append(key)
        self.values[key] = value

    def Get(self, key: int) -> int:
        return self.values.get(key, -1)
