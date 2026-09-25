//go:build linux

package keylogx

import (
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/titlebar"
)

const LocalKeyEventsPresent = true

const ButtonStyle = titlebar.ButtonStyleDefault

const BackgroundWorkerAllowed = false

func WaitForBackgroundEvents() (<-chan asig.Event, error) {
	panic("unimplemented")
}

func BackgroundEventsWriter() chan<- asig.Event {
	panic("unimplemented")
}

func SpawnBackgroundWorker() error {
	panic("unimplemented")
}
