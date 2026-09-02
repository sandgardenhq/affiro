//go:build darwin

package osxdriver

import (
	"fmt"
	"testing"
	"time"

	"github.com/sandgardenhq/affiro/asig"
)

func TestKeyMonitor(t *testing.T) {
	t.Skip("run manually for debugging only")
	t.Parallel()
	go func() {
		fmt.Println("Hey press a key!")
		for {
			ev, ok := Pop()
			if !ok {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			switch v := ev.(type) {
			case asig.MouseUpEvent:
				fmt.Println("mouse")
			case asig.KeyDownEvent:
				fmt.Printf("{%d,ctrl:%v,shift:%v,cmd:%v}\n",
					v.Key, v.ControlPressed, v.ShiftPressed, v.SpecialPressed)
			}
		}
		// 32000 == no key pressed
		// 0 = a
		// 9 = v
		// 1048840 = ctrl
		// 262401 = cmd
		// 131330 = shift
		// 524576 = option
		// 256 = no modifiers pressed
		// 1310985 ctrl + cmd
		// 1179914 ctrl + cmd + shift
		// 1966379 ctrl + cmd + shift + option
	}()
	StartKeyMonitor()
}
