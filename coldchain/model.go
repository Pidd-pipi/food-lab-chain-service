package coldchain

// Reading is a single temperature sample reported by a cold storage device.
type Reading struct {
	DeviceID string
	TempC    float64
	At       string
}

// Alert is an excursion notification for one device.
type Alert struct {
	DeviceID string
	Level    string
	Message  string
}

// Device is a registered cold storage unit.
type Device struct {
	ID     string
	Name   string
	Status string
}

// Policy holds the temperature thresholds for one device (or the fleet
// default). A reading outside the critical band is "critical", outside the
// warning band is "warning", otherwise "normal".
type Policy struct {
	WarningHigh  float64
	WarningLow   float64
	CriticalHigh float64
	CriticalLow  float64
}

func (p Policy) Level(tempC float64) string {
	if tempC >= p.CriticalHigh || tempC <= p.CriticalLow {
		return "critical"
	}
	if tempC >= p.WarningHigh || tempC <= p.WarningLow {
		return "warning"
	}
	return "normal"
}
