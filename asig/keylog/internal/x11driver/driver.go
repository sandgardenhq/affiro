//go:build linux

package x11driver

import (
	"fmt"
	"sync"

	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/keybind"
	"github.com/jezek/xgbutil/xevent"
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/asig/keylog/internal/x11driver/x11key"
	"golang.org/x/mobile/event/key"
)

// in x11 you have two options for how to respond to key events: you can grab the entire keyboard, or you can grab just some keys.
// it's a little more dangerous to grab the whole keyboard, but also far less intrusive (far fewer bindings).

var eventsMu sync.Mutex
var events []asig.Event

func StartKeyMonitor() error {
	xconn, err := xgbutil.NewConn()
	if err != nil {
		return fmt.Errorf("failed to init x11 conn: %w", err)
	}

	var keysyms x11key.KeysymTable

	const keyLo, keyHi = 8, 255
	km, err := xproto.GetKeyboardMapping(xconn.Conn(), keyLo, keyHi-keyLo+1).Reply()
	if err != nil {
		return fmt.Errorf("failed to get keyboard mapping: %w", err)
	}
	n := int(km.KeysymsPerKeycode)
	if n < 2 {
		return fmt.Errorf("too few keysyms per keycode found: %d", n)
	}
	for i := keyLo; i <= keyHi; i++ {
		keysyms[i][0] = uint32(km.Keysyms[(i-keyLo)*n+0])
		keysyms[i][1] = uint32(km.Keysyms[(i-keyLo)*n+1])
	}
	// note if we cared about numlock we'd need to do something tricky here, see shiny

	keybind.Initialize(xconn)

	keypressBinding := xevent.KeyPressFun(func(xu *xgbutil.XUtil, e xevent.KeyPressEvent) {
		r, c := keysyms.Lookup(uint8(e.Detail), e.State, 0)
		mods := x11key.KeyModifiers(e.State)
		eventsMu.Lock()
		var s string
		if r != 0 {
			s = string(r)
		}
		events = append([]asig.Event{
			asig.KeyDownEvent{
				Key:            c,
				String:         s,
				ControlPressed: mods&key.ModControl == key.ModControl,
				ShiftPressed:   mods&key.ModShift == key.ModShift,
				SpecialPressed: mods&key.ModMeta == key.ModMeta,
			},
		}, events...)
		eventsMu.Unlock()
		// Because all x11 keypress events go through us, we need to ask for the window in focus and
		// forward all events appropriately.
		focusWindowCookie := xproto.GetInputFocus(xconn.Conn())
		focusWindow, err := focusWindowCookie.Reply()
		if err != nil {
			fmt.Println("focus window check failed: " + err.Error())
		} else {
			e.Event = focusWindow.Focus
			e.Child = 0
			e.Time = xconn.TimeGet()
			ck := xproto.SendEventChecked(xconn.Conn(), false, focusWindow.Focus, xproto.EventMaskKeyPress, string(e.Bytes()))
			if err := ck.Check(); err != nil {
				fmt.Println("propagate send event failed: " + err.Error())
			}
		}
	})
	keypressBinding.Connect(xconn, xconn.RootWin())
	// This means: all keypress events for the whole x11 server now go through us.
	err = keybind.GrabKeyboard(xconn, xconn.RootWin())
	if err != nil {
		return err
	}

	// TODO: mousebinding breaks focus events i.e. the window in focus will always be the last window in focus, and focus can't change
	// because we've captured the mouse event; how can we listen to mouse events without breaking focus changing?

	// mousebind.Initialize(xconn)

	// mousepressBinding := mousebind.ButtonPressFun(func(xu *xgbutil.XUtil, e xevent.ButtonPressEvent) {
	// 	fmt.Println("test", e)
	// 	focusWindowCookie := xproto.GetInputFocus(xconn.Conn())
	// 	focusWindow, err := focusWindowCookie.Reply()
	// 	if err != nil {
	// 		fmt.Println("focus window check failed: " + err.Error())
	// 	} else {
	// 		e.Event = focusWindow.Focus
	// 		e.Child = 0
	// 		e.Time = xconn.TimeGet()
	// 		ck := xproto.SendEventChecked(xconn.Conn(), false, focusWindow.Focus, xproto.EventMaskButtonPress, string(e.Bytes()))
	// 		if err := ck.Check(); err != nil {
	// 			fmt.Println("propagate send event failed: " + err.Error())
	// 		}
	// 	}
	// })
	// mousepressBinding.Connect(xconn, xconn.RootWin(), "1", false, true)

	// mousereleaseBinding := mousebind.ButtonReleaseFun(func(xu *xgbutil.XUtil, e xevent.ButtonReleaseEvent) {
	// 	fmt.Println("test", e)
	// 	focusWindowCookie := xproto.GetInputFocus(xconn.Conn())
	// 	focusWindow, err := focusWindowCookie.Reply()
	// 	if err != nil {
	// 		fmt.Println("focus window check failed: " + err.Error())
	// 	} else {
	// 		e.Event = focusWindow.Focus
	// 		e.Child = 0
	// 		e.Time = xconn.TimeGet()
	// 		ck := xproto.SendEventChecked(xconn.Conn(), false, focusWindow.Focus, xproto.EventMaskButtonRelease, string(e.Bytes()))
	// 		if err := ck.Check(); err != nil {
	// 			fmt.Println("propagate send event failed: " + err.Error())
	// 		}
	// 	}
	// })
	// mousereleaseBinding.Connect(xconn, xconn.RootWin(), "1", false, true)

	xevent.Main(xconn)
	return nil
}

func Pop() (asig.Event, bool) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	if len(events) == 0 {
		return nil, false
	}
	ev := events[len(events)-1]
	events = events[:len(events)-1]
	return ev, true
}
