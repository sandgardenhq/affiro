//go:build windows

package windriver

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/Microsoft/go-winio"
	"github.com/oakmound/w32"
	"github.com/sandgardenhq/affiro/asig"
	"golang.org/x/mobile/event/key"
)

//go:embed dllmain/keylog.dll
var keylogDLL []byte
var keylog *syscall.DLL
var uninstall *syscall.Proc
var tmpDLLFilePath string
var closeListenerFunc func() error

// NB this must stay in sync with dllmain.c
const pipeName = "\\\\.\\pipe\\asig-keytracker"

func StartKeyMonitor() error {
	keyboardLayout = _GetKeyboardLayout(0)
	// TODO: monitor keyboard change events and update keyboard layout when they occur

	tmp := os.TempDir()
	fp := filepath.Join(tmp, "keylog.dll")
	if err := os.RemoveAll(fp); err != nil {
		fmt.Println(err)
	}
	if err := os.WriteFile(fp, keylogDLL, 0777); err != nil {
		return err
	}
	tmpDLLFilePath = fp
	var err error
	keylog, err = syscall.LoadDLL(fp)
	if err != nil {
		return err
	}

	pipeListener, err := winio.ListenPipe(pipeName, &winio.PipeConfig{
		InputBufferSize:  eventSize * 5000,
		OutputBufferSize: eventSize * 5000,
	})
	if err != nil {
		return err
	}
	closeListenerFunc = pipeListener.Close
	go func() {
		for {
			conn, err := pipeListener.Accept()
			if err != nil {
				if !errors.Is(err, winio.ErrPipeListenerClosed) {
					fmt.Println(err)
				}
				return
			}
			//fmt.Println("got connection")
			go handleConn(conn)
		}
	}()

	uninstall = keylog.MustFindProc("Uninstall")
	install := keylog.MustFindProc("Install")
	install.Call()
	//fmt.Println("called install")
	return nil
}

func Pop() (asig.Event, bool) {
	if len(eventQueue) == 0 {
		return nil, false
	}
	eventsMutex.Lock()
	next := eventQueue[0]
	eventQueue = eventQueue[1:]
	eventsMutex.Unlock()
	switch next.Type {
	case winHookTypeMouse:
		if next.WParam == w32.WM_LBUTTONDOWN || next.WParam == w32.WM_RBUTTONDOWN {
			return asig.MouseUpEvent{}, true
		}
	case winHookTypeKeyboard:
		isDown := false
		isRepeat := false
		switch next.Code {
		case 0:
			isDown = true
			//fmt.Print("down: ")
			const prevMask = 1 << 30
			if repeat := next.LParam&prevMask == prevMask; repeat {
				//	fmt.Print("repeat: ")
				isRepeat = true
			}
		default:
			fmt.Println("got unknown key code", next.Code, next.WParam, next.LParam)
			return nil, false
		}
		modifiers := keyModifiers()
		rn := readRune(uint32(next.WParam), uint8(next.LParam>>16))
		// TODO: this isn't a precise deduping, we're discarding repeat key presses
		if isDown && !isRepeat {
			return asig.KeyDownEvent{
				Key:            convVirtualKeyCode(uint32(next.WParam)),
				String:         string(rn),
				ControlPressed: modifiers&key.ModControl == key.ModControl,
				SpecialPressed: modifiers&key.ModMeta == key.ModMeta,
				ShiftPressed:   modifiers&key.ModShift == key.ModShift,
			}, true
		}
	}
	return nil, false
}

func Uninstall() uintptr {
	if closeListenerFunc != nil {
		//fmt.Println("closing listener")
		closeListenerFunc()
	}
	ret, _, _ := uninstall.Call()
	if err := keylog.Release(); err != nil {
		fmt.Println("failed to release dll: " + err.Error())
	}
	if err := os.Remove(tmpDLLFilePath); err != nil {
		fmt.Println("failed to remove dll: " + err.Error())
	}
	return ret
}

// NB: keep this in sync with dllmain.c
const winHookTypeKeyboard = 1
const winHookTypeMouse = 2

type winHookEvent struct {
	Type   uint64
	Code   uint64
	WParam uint64
	LParam uint64
}

const eventSize = 8 * 4

var eventsMutex sync.Mutex
var eventQueue []winHookEvent

func handleConn(c net.Conn) {
	defer c.Close()
	pBytes := make([]byte, eventSize)
	for {
		_, err := io.ReadFull(c, pBytes)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				//fmt.Println("EOF")
				return
			}
			fmt.Println("err reading", err)
			continue
		}
		p := winHookEvent{
			Type:   binary.LittleEndian.Uint64(pBytes[0:8]),
			Code:   binary.LittleEndian.Uint64(pBytes[8:16]),
			WParam: binary.LittleEndian.Uint64(pBytes[16:24]),
			LParam: binary.LittleEndian.Uint64(pBytes[24:32]),
		}
		//fmt.Println("payload:", p)
		eventsMutex.Lock()
		eventQueue = append(eventQueue, p)
		eventsMutex.Unlock()
	}
}
