package mathutil

import (
	"fmt"
	"math"
)

// SafeUint64ToInt converts uint64 to int with overflow checking.
// On overflow, returns the maximum int value and an error.
func SafeUint64ToInt(val uint64) (int, error) {
	// Check if value exceeds max int (platform-dependent)
	if val > math.MaxInt {
		return math.MaxInt, fmt.Errorf("overflow: uint64 value %d exceeds max int %d", val, math.MaxInt)
	}
	return int(val), nil
}

// SafeUint64ToInt64 converts uint64 to int64 with overflow checking.
// On overflow, returns the maximum int64 value and an error.
func SafeUint64ToInt64(val uint64) (int64, error) {
	if val > math.MaxInt64 {
		return math.MaxInt64, fmt.Errorf("overflow: uint64 value %d exceeds max int64 %d", val, math.MaxInt64)
	}
	return int64(val), nil
}

// Uint64ToInt converts uint64 to int, clamping to max int on overflow.
// This is safe for display/formatting purposes where exact value isn't critical.
func Uint64ToInt(val uint64) int {
	if val > math.MaxInt {
		return math.MaxInt
	}
	return int(val)
}

// Uint64ToInt64 converts uint64 to int64, clamping to max int64 on overflow.
// This is safe for display/formatting purposes where exact value isn't critical.
func Uint64ToInt64(val uint64) int64 {
	if val > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(val)
}
