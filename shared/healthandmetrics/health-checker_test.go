package healthandmetrics

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type healthCheckObj string

func (h healthCheckObj) HealthCheckAddress() string {
	return string(h)
}

type healthCheckProcessorMock struct {
	err        error
	delay      time.Duration
	checkCount int
	m          sync.Mutex
}

func (h *healthCheckProcessorMock) Check(object HealthCheckObject) error {
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	h.m.Lock()
	h.checkCount++
	h.m.Unlock()
	return h.err
}

func (h *healthCheckProcessorMock) checksCount() int {
	h.m.Lock()
	defer h.m.Unlock()
	return h.checkCount
}

func Test_healthChecker_StartSkippingDelay(t *testing.T) {
	proc := healthCheckProcessorMock{
		err:   nil,
		delay: time.Millisecond * 5,
	}

	obj := healthCheckObj("127.0.0.1:9000")
	checker := NewHealthChecker(time.Millisecond*4, 1, 1, &proc)
	checker.AddHealthCheckObject(obj)
	go checker.Start()

	time.Sleep(time.Millisecond * 13)

	assert.Equal(t, 2, proc.checksCount())
}

func Test_healthChecker_StartRespectDelay(t *testing.T) {
	proc := healthCheckProcessorMock{
		err:   nil,
		delay: time.Millisecond * 5,
	}

	obj := healthCheckObj("127.0.0.1:9000")
	checker := NewHealthChecker(time.Millisecond*10, 1, 1, &proc)
	checker.AddHealthCheckObject(obj)
	go checker.Start()

	time.Sleep(time.Millisecond * 13)

	assert.Equal(t, 1, proc.checksCount())
}

// waitForChecksAtLeast polls until the mock has recorded at least n checks,
// failing the test if that doesn't happen within a generous timeout. This
// keeps the test deterministic instead of racing against wall-clock sleeps.
func waitForChecksAtLeast(t *testing.T, proc *healthCheckProcessorMock, n int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if proc.checksCount() >= n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d checks, got %d", n, proc.checksCount())
}

func Test_healthChecker_ToleratesTransientFailuresBelowThreshold(t *testing.T) {
	proc := healthCheckProcessorMock{
		err: fmt.Errorf("connection refused"),
	}

	obj := healthCheckObj("127.0.0.1:9000")
	checker := NewHealthChecker(time.Millisecond*20, 1, 3, &proc)
	checker.AddHealthCheckObject(obj)

	var failedMu sync.Mutex
	var failedCount int
	checker.AddFailedObserver(func(HealthCheckObject, error) {
		failedMu.Lock()
		failedCount++
		failedMu.Unlock()
	})

	go checker.Start()

	// After 2 failed sweeps we're still below the threshold of 3, so the
	// object must survive.
	waitForChecksAtLeast(t, &proc, 2)
	time.Sleep(time.Millisecond * 5)
	failedMu.Lock()
	assert.Equal(t, 0, failedCount)
	failedMu.Unlock()

	// The 3rd failed sweep crosses the threshold and evicts the object.
	waitForChecksAtLeast(t, &proc, 3)
	time.Sleep(time.Millisecond * 5)
	failedMu.Lock()
	assert.Equal(t, 1, failedCount)
	failedMu.Unlock()
}

func Test_healthChecker_SuccessResetsFailureCounter(t *testing.T) {
	proc := healthCheckProcessorMock{
		err: fmt.Errorf("connection refused"),
	}

	obj := healthCheckObj("127.0.0.1:9000")
	checker := NewHealthChecker(time.Millisecond*20, 1, 2, &proc)
	checker.AddHealthCheckObject(obj)

	var failedMu sync.Mutex
	var failedCount int
	checker.AddFailedObserver(func(HealthCheckObject, error) {
		failedMu.Lock()
		failedCount++
		failedMu.Unlock()
	})

	go checker.Start()

	// One failed sweep, then flip to success for one sweep, then fail again:
	// the intervening success must reset the counter so the two isolated
	// failures never combine to cross the threshold of 2.
	waitForChecksAtLeast(t, &proc, 1)
	proc.m.Lock()
	proc.err = nil
	proc.m.Unlock()

	waitForChecksAtLeast(t, &proc, 2)
	proc.m.Lock()
	proc.err = fmt.Errorf("connection refused")
	proc.m.Unlock()

	waitForChecksAtLeast(t, &proc, 3)
	time.Sleep(time.Millisecond * 5)
	failedMu.Lock()
	assert.Equal(t, 0, failedCount, "isolated single failures should never accumulate across a success")
	failedMu.Unlock()
}

func Test_healthChecker_StartWithAddingObjectsAfterStart(t *testing.T) {
	proc := healthCheckProcessorMock{
		err:   nil,
		delay: time.Millisecond * 1,
	}

	obj := healthCheckObj("127.0.0.1:9000")
	checker := NewHealthChecker(time.Millisecond*5, 1, 1, &proc)
	go checker.Start()
	time.Sleep(time.Millisecond * 1)

	checker.AddHealthCheckObject(obj)
	time.Sleep(time.Millisecond * 8)

	assert.Equal(t, 1, proc.checksCount())
}
