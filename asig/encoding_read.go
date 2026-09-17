package asig

import (
	"bufio"
	"fmt"
	"io"
)

// Read retrieves a string signature from a stream; it should be the next token in the stream
// and a newline should mark its completion
func Read(r io.Reader) (signature *Asig, remaining *bufio.Reader, err error) {
	buff := bufio.NewReader(r)
	candidate, err := buff.ReadString('\n')
	if err != nil {
		return nil, buff, fmt.Errorf("reading up to the end of the signature: %w", err)
	}
	signature, err = ParseString(candidate)
	return signature, buff, err
}
