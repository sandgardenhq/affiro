//go:build !darwin && !linux && !windows

package keylog

import "errors"

func Start() (Monitor, error) {
	return nil, errors.New("unimplemented")
}
