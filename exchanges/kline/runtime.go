package kline

import (
	"errors"
	"fmt"
	"unsafe"
)

var table unsafe.Pointer

func Wow(check Interval) (Interval, error) {
	// var prev Interval
	for x := range SupportedIntervals {
		if SupportedIntervals[x] <= check {
			for y := x; y != -1; y-- {
				fmt.Printf("Checking %s -> %s\n", check, SupportedIntervals[y])
				if check%SupportedIntervals[y] == 0 {
					return SupportedIntervals[y], nil
				}
			}
		}
	}
	return 0, errors.New("broken bro")
}
