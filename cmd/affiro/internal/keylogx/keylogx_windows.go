//go:build windows

package keylogx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/oakmound/oak/v4/key"
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/titlebar"
)

const LocalKeyEventsPresent = true

const ButtonStyle = titlebar.ButtonStyleDefault

const BackgroundWorkerAllowed = true

const backgroundPipeName = "\\\\.\\pipe\\asig-background"

func WaitForBackgroundEvents() (<-chan asig.Event, error) {
	pipeListener, err := winio.ListenPipe(backgroundPipeName, &winio.PipeConfig{
		InputBufferSize:  eventSize * 5000,
		OutputBufferSize: eventSize * 5000,
	})
	if err != nil {
		return nil, err
	}
	ch := make(chan asig.Event, 20)
	go func() {
		for {
			conn, err := pipeListener.Accept()
			if err != nil {
				if !errors.Is(err, winio.ErrPipeListenerClosed) {
					fmt.Println(err)
				}
				return
			}
			go handleConn(conn, ch)
		}
	}()
	return ch, nil
}

func handleConn(c net.Conn, ch chan<- asig.Event) {
	defer c.Close()
	pBytes := make([]byte, eventSize)
	for {
		_, err := io.ReadFull(c, pBytes)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// fmt.Println("EOF")
				return
			}
			fmt.Println("err reading", err)
			continue
		}
		ch <- bytesToEvent(pBytes)
	}
}

type binaryEventEncoding struct {
	str     [4]byte
	code    uint32
	control byte
	special byte
	shift   byte
	mouse   byte
}

const eventSize = 12

func BackgroundEventsWriter() chan<- asig.Event {
	ch := make(chan asig.Event, 20)
	go func() {
		for {
			ev := <-ch
			pipe, err := winio.DialPipe(backgroundPipeName, nil)
			if err != nil {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			buff := make([]byte, eventSize)
			switch v := ev.(type) {
			case asig.MouseUpEvent:
				buff = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
			case asig.KeyDownEvent:
				strBytes := strTo4Bytes(v.String)
				copy(buff, strBytes[:])
				binary.LittleEndian.PutUint32(buff[4:], uint32(v.Key))
				buff[8] = boolToByte(v.ControlPressed)
				buff[9] = boolToByte(v.SpecialPressed)
				buff[10] = boolToByte(v.ShiftPressed)
			}
			n, err := pipe.Write(buff)
			if err != nil {
				fmt.Printf("failed to write event: %v\n", err)
				continue
			}
			if n != eventSize {
				fmt.Printf("short write: expected %v got %v\n", eventSize, n)
				continue
			}
		}
	}()
	return ch
}

func strTo4Bytes(s string) [4]byte {
	b := []byte(s)
	if len(b) > 4 {
		return [4]byte(b[:4])
	}
	b4 := [4]byte{}
	for i := 0; i < 4; i++ {
		if i >= len(b) {
			break
		}
		b4[i] = b[i]
	}
	return b4
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func bytesToEvent(pBytes []byte) asig.Event {
	p := binaryEventEncoding{
		str:     [4]byte(pBytes[0:4]),
		code:    binary.LittleEndian.Uint32(pBytes[4:8]),
		control: pBytes[8],
		special: pBytes[9],
		shift:   pBytes[10],
		mouse:   pBytes[11],
	}
	if p.mouse == 1 {
		return asig.MouseUpEvent{}
	}
	return asig.KeyDownEvent{
		String:         string(p.str[:]),
		Key:            key.Code(p.code),
		ControlPressed: p.control == 1,
		ShiftPressed:   p.shift == 1,
		SpecialPressed: p.special == 1,
	}
}

func SpawnBackgroundWorker() error {
	cmd := exec.Command(os.Args[0], "--background-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Process.Release()
}
