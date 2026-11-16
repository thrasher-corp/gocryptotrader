//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will run the chain
// analysis tests in bitflyer_test.go against the live API
package bitflyer

var mockTests = false
