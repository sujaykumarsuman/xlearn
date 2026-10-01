class MinStack {
public:
    MinStack() {}

    void push(int val) {
        vals.push_back(val);
        mins.push_back(mins.empty() ? val : min(val, mins.back()));
    }

    void pop() {
        vals.pop_back();
        mins.pop_back();
    }

    int top() { return vals.back(); }

    int getMin() { return mins.back(); }

private:
    vector<int> vals, mins;
};
