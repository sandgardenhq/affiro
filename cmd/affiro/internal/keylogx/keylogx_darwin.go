//go:build darwin

package keylogx

import (
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/titlebar"
)

const LocalKeyEventsPresent = false

const ButtonStyle = titlebar.ButtonStyleOSX

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
