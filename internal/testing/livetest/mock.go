//go:build !mock_test_off

package livetest

// Mock reports whether tests stand in for the live APIs they exercise. Build with the mock_test_off tag to call
// those APIs live.
const Mock = true
