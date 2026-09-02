//go:build !linux && !darwin && !windows

package keylogx

import "github.com/sandgardenhq/affiro/cmd/affiro/internal/titlebar"

const LocalKeyEventsPresent = false

const ButtonStyle = titlebar.ButtonStyleDefault
