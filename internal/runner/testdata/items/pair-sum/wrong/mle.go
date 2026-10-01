// want: MLE
package main

var sink []byte

func pairSum(nums []int, target int) []int {
	sink = make([]byte, 1<<30)
	for i := range sink {
		sink[i] = 1
	}
	return nil
}
