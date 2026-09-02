//go:build windows

package windriver

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestKeyMonitor(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	c := make(chan os.Signal, 10)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	err := StartKeyMonitor()
	if err != nil {
		t.Fatal(err)
	}
	timeout := time.After(100 * time.Second)
	for {
		select {
		case <-timeout:
			fmt.Println("timeout, uninstall")
			Uninstall()
			return
		case <-c:
			fmt.Println("ctrl+c, uninstall")
			Uninstall()
			return
		default:
		}
		v, ok := Pop()
		if ok {
			fmt.Println(v)
		}
	}
}
