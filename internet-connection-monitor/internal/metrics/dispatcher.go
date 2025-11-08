package metrics

import (
	"sync"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// Dispatcher distributes test results to all output modules
type Dispatcher struct {
	outputs []Output
	mu      sync.RWMutex
}

// Output is an interface for result output modules
type Output interface {
	// Write sends a test result to the output
	Write(result *models.TestResult) error

	// Name returns the output module name
	Name() string
}

// NewDispatcher creates a new result dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		outputs: make([]Output, 0),
	}
}

// RegisterOutput adds an output module to the dispatcher
func (d *Dispatcher) RegisterOutput(output Output) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.outputs = append(d.outputs, output)
}

// Dispatch sends a result to all registered outputs
// Outputs are called in parallel to avoid blocking
func (d *Dispatcher) Dispatch(result *models.TestResult) {
	d.mu.RLock()
	outputs := make([]Output, len(d.outputs))
	copy(outputs, d.outputs)
	d.mu.RUnlock()

	// Fan out to all outputs in parallel
	var wg sync.WaitGroup
	for _, output := range outputs {
		wg.Add(1)
		go func(o Output) {
			defer wg.Done()
			if err := o.Write(result); err != nil {
				// TODO: Log error (but don't fail the dispatch)
				// We don't want one failing output to block others
			}
		}(output)
	}

	// Wait for all outputs to complete
	wg.Wait()
}
