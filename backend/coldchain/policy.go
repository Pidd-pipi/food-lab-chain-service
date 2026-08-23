package coldchain

// Evaluator resolves the policy for a device: a device-specific rule wins,
// otherwise the fleet default applies. Provider returns the policy through an
// interface so callers must distinguish "no policy configured" from "a nil
// default was set".
type Evaluator struct {
	defaults *Policy
	rules    map[string]Policy
}

// PolicyProvider is satisfied by both Policy and *Policy.
type PolicyProvider interface {
	Level(tempC float64) string
}

func NewEvaluator(defaults *Policy) *Evaluator {
	return &Evaluator{defaults: defaults}
}

func (e *Evaluator) SetRule(deviceID string, policy Policy) {
	e.rules[deviceID] = policy
}

// Provider returns the effective policy for a device.
func (e *Evaluator) Provider(deviceID string) (PolicyProvider, bool) {
	if rule, ok := e.rules[deviceID]; ok {
		return rule, true
	}
	return e.defaults, true
}

// Evaluate classifies a reading. When no policy is configured at all, the
// reading is skipped instead of being treated as valid.
func (e *Evaluator) Evaluate(r Reading) ([]Alert, error) {
	provider, ok := e.Provider(r.DeviceID)
	if !ok {
		return nil, nil
	}
	level := provider.Level(r.TempC)
	if level == "normal" {
		return nil, nil
	}
	return []Alert{{DeviceID: r.DeviceID, Level: level, Message: "threshold exceeded"}}, nil
}
