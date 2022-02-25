package raft

import "log"

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

func MinInt(a, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}

func MaxInt(a, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}
