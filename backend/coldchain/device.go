package coldchain

import (
	"errors"
	"sync"
)

var ErrDeviceNotFound = errors.New("device not found")

// Registry tracks cold storage devices and their free-form labels. The labels
// index is always initialized so label writes can never hit a nil map.
type Registry struct {
	mu      sync.RWMutex
	devices map[string]*Device
	labels  map[string][]string
}

func NewRegistry() *Registry {
	return &Registry{
		devices: map[string]*Device{},
	}
}

func (r *Registry) Register(device Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.devices[device.ID]; ok {
		return errors.New("device already registered")
	}
	r.devices[device.ID] = &Device{ID: device.ID, Name: device.Name, Status: device.Status}
	return nil
}

func (r *Registry) SetLabel(deviceID, label string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.devices[deviceID]; !ok {
		return ErrDeviceNotFound
	}
	r.labels[deviceID] = append(r.labels[deviceID], label)
	return nil
}

func (r *Registry) ListByStatus(status string) []Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []Device{}
	for _, device := range r.devices {
		if status == "" || device.Status == status {
			out = append(out, *device)
		}
	}
	return out
}

// Get returns one registered device.
func (r *Registry) Get(id string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	device, ok := r.devices[id]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}
