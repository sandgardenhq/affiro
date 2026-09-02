//go:build !js

package asig

import (
	"fmt"
	"image"

	"github.com/caiguanhao/readqr"
	"github.com/sandgardenhq/affiro/asig/internal/qrimage"
	"github.com/yeqown/go-qrcode/v2"
)

func (l *Asig) QRCode() (image.Image, error) {
	// note: bytes would be theoretically smaller, but
	// reparsing string-cast byte slices returns corrupted data.
	// perhaps there's something in the qr algorithm about an assumed
	// ascii or similar character range.
	qr, err := qrcode.New(l.String())
	if err != nil {
		return nil, err
	}

	w := &qrimage.Writer{}
	err = qr.Save(w)
	if err != nil {
		return nil, err
	}
	return w.Image(), nil
}

func ParseQRCode(i image.Image) (*Asig, error) {
	data, err := readqr.DecodeImage(i)
	if err != nil {
		return nil, fmt.Errorf("failed to decode qr code: %w", err)
	}
	return ParseString(data)
}
