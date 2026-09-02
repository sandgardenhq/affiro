package random

import (
	"crypto/rand"
	"math/big"
)

// String returns a string from alphanumeric characters of length n.
func String(n int) string {
	return str(n, strChars)
}

var strChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

// UpperString returns a string from only uppercase english alphabet characters of length n.
func UpperString(n int) string {
	return str(n, upperStrChars)
}

var upperStrChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func str(n int, charset []byte) string {
	s := make([]byte, n)
	maxInt := big.NewInt(int64(len(charset)))
	for i := range n {
		passIndex, err := rand.Int(rand.Reader, maxInt)
		if err != nil {
			panic(err)
		}
		s[i] = charset[passIndex.Int64()]
	}
	return string(s)
}
