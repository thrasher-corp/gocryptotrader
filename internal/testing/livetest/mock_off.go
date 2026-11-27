//go:build mock_test_off

package livetest

// Mock is false under the mock_test_off tag, so tests call the live APIs they otherwise stand in for
const Mock = false
