package coldchain

import "context"

// Report is the daily excursion summary for one device.
type Report struct {
	DeviceID   string
	Date       string
	Readings   int
	Excursions int
	MaxC       float64
	MinC       float64
}

// BuildReport samples the device history for the day and combines it with the
// excursion counter.
func BuildReport(ctx context.Context, monitor *Monitor, deviceID, date string) (*Report, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	ring, err := monitor.Recent(ctx, deviceID, 0)
	if err != nil {
		return nil, err
	}
	report := &Report{DeviceID: deviceID, Date: date, MinC: 9999}
	for _, reading := range ring {
		report.Readings++
		if reading.TempC > report.MaxC {
			report.MaxC = reading.TempC
		}
		if reading.TempC < report.MinC {
			report.MinC = reading.TempC
		}
	}
	report.Excursions = monitor.Excursions(deviceID)
	return report, nil
}

// Summarize rolls the per-device reports into a device -> excursion count map.
// The map is always allocated, even for an empty input, so callers can write
// into it without panicking.
func Summarize(ctx context.Context, reports []*Report) (map[string]int, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if len(reports) == 0 {
		return nil, nil
	}
	out := map[string]int{}
	for _, report := range reports {
		if report == nil {
			continue
		}
		out[report.DeviceID] = report.Excursions
	}
	return out, nil
}
