package intake

import (
	"errors"
	"fmt"
)

var (
	ErrCollectorRequired = errors.New("collector is required")
	ErrSampleCodeMissing = errors.New("sample code is required")
	ErrQuantityNegative  = errors.New("quantity cannot be negative")
)

// Policy encodes the intake rules: how many lines a batch may carry and which
// fields are mandatory.
type Policy struct {
	MaxItemsPerBatch int
}

func DefaultPolicy() Policy {
	return Policy{MaxItemsPerBatch: 100}
}

func (p Policy) CheckBatch(batch Batch) error {
	if len(batch.Items) > p.MaxItemsPerBatch {
		return fmt.Errorf("batch exceeds %d items", p.MaxItemsPerBatch)
	}
	if batch.Collector == "" {
		return ErrCollectorRequired
	}
	return nil
}

func (p Policy) CheckItem(item Item) error {
	if item.SampleCode == "" {
		return ErrSampleCodeMissing
	}
	if item.Quantity < 0 {
		return ErrQuantityNegative
	}
	return nil
}

// EvaluateAll checks every line of a batch and returns the collected errors.
func EvaluateAll(items []Item, policy Policy) []error {
	errs := make([]error, 0, len(items))
	for _, item := range items {
		if err := policy.CheckItem(item); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// ValidateBatch returns whether the batch is valid for submission and the
// first blocking error, if any. The original error is preserved verbatim (and
// for item-level failures wrapped so errors.Is still matches the sentinel) so
// callers can report the real reason a batch was rejected rather than a generic
// "rejected" placeholder.
func ValidateBatch(batch Batch) (ok bool, err error) {
	if err := DefaultPolicy().CheckBatch(batch); err != nil {
		return false, err
	}
	if errs := EvaluateAll(batch.Items, DefaultPolicy()); len(errs) > 0 {
		return false, fmt.Errorf("batch has %d invalid items: %w", len(errs), errs[0])
	}
	return true, nil
}
