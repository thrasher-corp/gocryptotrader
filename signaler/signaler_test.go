package signaler

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// mockSignalNotifier is a mock implementation for testing
type mockSignalNotifier struct {
	ch chan os.Signal
}

// newMockSignalNotifier creates a new mock for testing
func newMockSignalNotifier() *mockSignalNotifier {
	return &mockSignalNotifier{
		ch: make(chan os.Signal, 10),
	}
}

// Notify starts listening (in the mock, we just store the channel)
func (m *mockSignalNotifier) Notify(c chan<- os.Signal, _ ...os.Signal) {
	go func() {
		for signal := range m.ch {
			c <- signal
		}
	}()
}

// Stop stops listening (in the mock, we close our internal channel)
func (m *mockSignalNotifier) Stop(_ chan<- os.Signal) {
	close(m.ch)
}

// SendSignal is a test helper - let us fake sending signal
func (m *mockSignalNotifier) SendSignal(sig os.Signal) {
	select {
	case m.ch <- sig:
		// succesffully sent
	default:
		// Channel is full or closed, ignore silently
	}
}

// setNotifierForTesting allows to inject a mock notifier
// This should only be used in tests
func setNotifierForTesting(n SignalNotifier) {
	notifier = n
}

func TestMockSignalNotifier(t *testing.T) {
	// Create a test channel
	testChannel := make(chan os.Signal, 1)

	// Create a mock notifier
	mock := newMockSignalNotifier()

	mock.Notify(testChannel, os.Interrupt)

	// send fake signal
	go func() {
		time.Sleep(400 * time.Millisecond)
		mock.SendSignal(os.Interrupt)
	}()

	// Wait for signal (just like WaitForInterrupt)
	select {
	case sig := <-testChannel:
		if sig != os.Interrupt {
			t.Errorf("Expected %v, got %v", os.Interrupt, sig)
		}
		t.Logf("Successfully received fake signal: %v", sig)
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for signal")
	}
}

func TestWaitForInterrupt(t *testing.T) {
	originalNotifier := notifier
	defer func() {
		notifier = originalNotifier
		s = make(chan os.Signal, 1)
		sigs := getPlatformSignals()
		notifier.Notify(s, sigs...)
	}()

	// Stop the real OS signal handler first
	originalNotifier.Stop(s)

	// create and inject mock
	mock := newMockSignalNotifier()
	setNotifierForTesting(mock)

	// we need to reinitialize with the mock
	// Clear the old channel and create a new one
	s = make(chan os.Signal, 1)
	sigs := getPlatformSignals()
	notifier.Notify(s, sigs...)

	// test the actual WaitForInterrupt function
	go func() {
		time.Sleep(10 * time.Millisecond)
		mock.SendSignal(os.Interrupt)
	}()

	sig := WaitForInterrupt()
	if sig != os.Interrupt {
		t.Errorf("Expected %v, got %v", os.Interrupt, sig)
	}
	t.Logf("Successfully received fake signal: %v", sig)
}

func TestGetPlatforms(t *testing.T) {
	signals := getPlatformSignals()

	requiredSignals := []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGABRT,
	}

	for _, required := range requiredSignals {
		found := false
		for _, actual := range signals {
			if actual == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required signal %v missing from platform signals", required)
		}
	}

	hasKill := false
	for _, signal := range signals {
		if signal == os.Kill {
			hasKill = true
		}
	}

	// check platform-specific signal
	if runtime.GOOS == "windows" {
		if hasKill {
			t.Logf("os.Kill is available on windows")
		} else {
			t.Logf("os.Kill is not available on windows")
		}
	} else {
		if hasKill {
			t.Errorf("os.Kill should not be included on %s (cannot be caught)", runtime.GOOS)
		} else {
			t.Logf("os.Kill is correctly excluded on %s", runtime.GOOS)
		}
	}

	t.Logf("Platform: %s, Signals: %v", runtime.GOOS, signals)
}

func TestSignalNotifierInterface(t *testing.T) {
	// Test if both implementations satisfy the interface
	var _ SignalNotifier = &osSignalNotifier{}
	var _ SignalNotifier = &mockSignalNotifier{}
	t.Logf("osSignalNotifier and mockSignalNotifier implement SignalNotifier interface")
}

func TestMockSignalNotifierStop(t *testing.T) {
	testChannel := make(chan os.Signal, 1)
	mock := newMockSignalNotifier()

	mock.Notify(testChannel, os.Interrupt)

	mock.SendSignal(os.Interrupt)

	// should receive it
	select {
	case sig := <-testChannel:
		if sig != os.Interrupt {
			t.Errorf("Expected %v, got %v", os.Interrupt, sig)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for signal")
	}

	mock.Stop(testChannel)
}

func TestMultipleSignalTypes(t *testing.T) {
	originalNotifier := notifier
	defer func() {
		notifier = originalNotifier
		s = make(chan os.Signal, 1)
		sigs := getPlatformSignals()
		notifier.Notify(s, sigs...)
	}()

	originalNotifier.Stop(s)

	mock := newMockSignalNotifier()
	setNotifierForTesting(mock)

	s = make(chan os.Signal, 1)
	sigs := getPlatformSignals()
	notifier.Notify(s, sigs...)

	testSignals := []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGABRT}

	for _, testSig := range testSignals {
		t.Run(testSig.String(), func(t *testing.T) {
			go func() {
				time.Sleep(10 * time.Millisecond)
				mock.SendSignal(testSig)
			}()

			sig := WaitForInterrupt()
			if sig != testSig {
				t.Errorf("Expected %v, got %v", testSig, sig)
			}
			t.Logf("Successfully received %v", testSig)
		})
	}
}
