package asig

import (
	"bufio"
	"io"
)

// Read retrieves a string signature from a stream; it should be the next token in the stream
// and a newline should mark its completion
func Read(r io.Reader) (signature *Asig, remaining *bufio.Reader, err error) {
	buff := bufio.NewReader(r)
	candidate, err := buff.ReadString('\n')
	if err != nil {
		return nil, buff, err
	}
	signature, err = ParseString(string(candidate))
	return signature, buff, err
}
