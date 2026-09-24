//go:build !darwin && !linux && !windows

package keylog

func NewMonitor() Monitor {
	panic("unimplemented")
}
