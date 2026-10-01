class FixedShelf {
public:
    FixedShelf(int capacity) : capacity_(capacity) {}

    void Put(int key, int value) {
        auto it = values_.find(key);
        if (it != values_.end()) {
            it->second = value;
            return;
        }
        if ((int)order_.size() == capacity_) {
            values_.erase(order_.front());
            order_.pop_front();
        }
        order_.push_back(key);
        values_[key] = value;
    }

    int Get(int key) {
        auto it = values_.find(key);
        return it == values_.end() ? -1 : it->second;
    }

private:
    int capacity_;
    unordered_map<int, int> values_;
    deque<int> order_;
};
