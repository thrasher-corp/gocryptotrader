//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will mock the
// chain analysis tests in bitflyer_test.go
package bitflyer

var mockTests = true
