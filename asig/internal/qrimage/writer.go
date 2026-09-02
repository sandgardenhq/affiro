package qrimage

import (
	"errors"
	"image"
	"image/color"
	"log"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard/imgkit"

	"github.com/fogleman/gg"
)

var _ qrcode.Writer = &Writer{}

var (
	ErrNilWriter = errors.New("nil writer")
)

// Writer is a writer that writes QR Code to an image.Image.
type Writer struct {
	option *outputImageOptions
	image  image.Image
}

// New creates a standard writer.
func New(opts ...ImageOption) *Writer {
	dst := defaultOutputImageOption()
	for _, opt := range opts {
		opt.apply(dst)
	}

	return &Writer{
		option: dst,
	}
}

const (
	_defaultPadding = 40
)

func (w *Writer) Image() image.Image {
	return w.image
}

func (w *Writer) Write(mat qrcode.Matrix) error {
	w.image = drawTo(mat, w.option)
	return nil
}

func (w *Writer) Close() error {
	return nil
}

func (w *Writer) Attribute(dimension int) *Attribute {
	return w.option.preCalculateAttribute(dimension)
}

func drawTo(mat qrcode.Matrix, option *outputImageOptions) image.Image {
	if option == nil {
		option = defaultOutputImageOption()
	}

	return draw(mat, option)
}

// draw deal QRCode's matrix to be an image.Image. Notice that if anyone changed this function,
// please also check the function outputImageOptions.preCalculateAttribute().
func draw(mat qrcode.Matrix, opt *outputImageOptions) image.Image {
	top, right, bottom, left := opt.borderWidths[0], opt.borderWidths[1], opt.borderWidths[2], opt.borderWidths[3]
	// closer as image width, h as image height
	w := mat.Width()*opt.qrBlockWidth() + left + right
	h := mat.Height()*opt.qrBlockWidth() + top + bottom
	dc := gg.NewContext(w, h)

	// draw background
	dc.SetColor(opt.backgroundColor())
	dc.DrawRectangle(0, 0, float64(w), float64(h))
	dc.Fill()

	// qrcode block draw context
	ctx := &DrawContext{
		Context: dc,
		x:       0.0,
		y:       0.0,
		w:       opt.qrBlockWidth(),
		h:       opt.qrBlockWidth(),
		color:   color.Black,
	}
	shape := opt.getShape()

	var (
		halftoneImg image.Image
		halftoneW   = float64(opt.qrBlockWidth()) / 3.0
	)
	if opt.halftoneImg != nil {
		halftoneImg = imgkit.Binaryzation(
			imgkit.Scale(opt.halftoneImg, image.Rect(0, 0, mat.Width()*3, mat.Width()*3), nil),
			60,
		)

		// _ = imgkit.Save(halftoneImg, "mask.jpeg")
	}

	// Check if the logo image exists and fits within the QR code bounds.
	// If so, mark it as valid and store its dimensions for safe zone calculation.
	var logoValid bool
	var logoWidth, logoHeight int
	if opt.logoImage() != nil {
		bound := opt.logo.Bounds()
		upperLeft, lowerRight := bound.Min, bound.Max
		logoWidth, logoHeight = lowerRight.X-upperLeft.X, lowerRight.Y-upperLeft.Y

		logoValid = validLogoImage(w, h, logoWidth, logoHeight, opt.logoSizeMultiplier)
	}

	// bitMap stores which blocks are set (true = active block)
	bitMap := mat.Bitmap()
	// If the logo safe zone is enabled, clear the corresponding area in bitMap
	if logoValid && opt.logoSafeZone {
		mat.Iterate(qrcode.IterDirection_ROW, func(x int, y int, v qrcode.QRValue) {
			if blockOverlapsLogo(x, y, opt.qrBlockWidth(), left, top, w, h, logoWidth, logoHeight) {
				bitMap[x][y] = false
			}
		})
	}

	// iterate the matrix to Draw each pixel
	mat.Iterate(qrcode.IterDirection_ROW, func(x int, y int, v qrcode.QRValue) {
		// Skip drawing this block if it overlaps with the logo area.
		// This preserves logo visibility by preventing block rendering underneath it.
		if logoValid && opt.logoSafeZone &&
			blockOverlapsLogo(x, y, opt.qrBlockWidth(), left, top, w, h, logoWidth, logoHeight) {
			if v.IsSet() {
				return
			}
		}

		// Draw the block
		ctx.x, ctx.y = float64(x*opt.qrBlockWidth()+left), float64(y*opt.qrBlockWidth()+top)
		ctx.w, ctx.h = opt.qrBlockWidth(), opt.qrBlockWidth()
		ctx.color = opt.translateToRGBA(v)
		ctx.neighbours = getNeighbours(bitMap, x, y)

		// DONE(@yeqown): make this abstract to Shapes
		switch typ := v.Type(); typ {
		case qrcode.QRType_FINDER:
			shape.DrawFinder(ctx)
		case qrcode.QRType_DATA:
			if halftoneImg == nil {
				shape.Draw(ctx)
				return
			}

			ctx2 := &DrawContext{
				Context: ctx.Context,
				w:       int(halftoneW),
				h:       int(halftoneW),
			}
			// only halftone image enabled and current block is Data.
			for i := range 3 {
				for j := range 3 {
					ctx2.x, ctx2.y = ctx.x+float64(i)*halftoneW, ctx.y+float64(j)*halftoneW
					if i == 1 && j == 1 {
						ctx2.color = ctx.color
					} else {
						ctx2.color = halftoneColor(halftoneImg, opt.bgTransparent, x*3+i, y*3+j)
					}
					shape.Draw(ctx2)
				}
			}
		default:
			shape.Draw(ctx)
		}

		// EOFn
	})

	// Gradient fill
	if opt.qrGradient != nil {
		img := opt.qrGradient.applyGradient(dc.Image(), opt.qrColor)
		dc.DrawImage(img, 0, 0)
	}

	if opt.logoImage() == nil {
		goto done
	}

	// Log a warning and skip drawing the logo if it exceeds the allowed size ratio.
	if !logoValid {
		log.Printf("w=%d, h=%d, logoW=%d, logoH=%d, logo is over than 1/%d of QRCode \n",
			w, h, logoWidth, logoHeight, opt.logoSizeMultiplier)
		goto done
	}

	// DONE(@yeqown): calculate the xOffset and yOffset which point(xOffset, yOffset)
	// should icon upper-left to start
	dc.DrawImage(opt.logoImage(), (w-logoWidth)/2, (h-logoHeight)/2)

done:
	return dc.Image()
}

// getNeighbours returns a bitmask (uint16) representing the 8 neighboring cells
// around the (x, y) position in the matrix. Each bit corresponds to a specific
// direction and is set if the neighboring cell is within bounds and set to `true`.
// The center cell itself (x, y) is included as NSelf if it is also `true`.
func getNeighbours(mtx [][]bool, x, y int) uint16 {
	dirs := []struct {
		dx, dy int
		flag   uint16
	}{
		{-1, -1, NTopLeft},
		{0, -1, NTop},
		{1, -1, NTopRight},
		{-1, 0, NLeft},
		{1, 0, NRight},
		{-1, 1, NBotLeft},
		{0, 1, NBot},
		{1, 1, NBotRight},
	}

	var res uint16

	rows := len(mtx)
	if rows == 0 {
		return res
	}
	cols := len(mtx[0])

	// Check the center cell
	if y >= 0 && y < rows && x >= 0 && x < cols && mtx[y][x] {
		res |= NSelf
	}

	// Check neighbors
	for _, d := range dirs {
		nx, ny := x+d.dx, y+d.dy
		if ny >= 0 && ny < rows && nx >= 0 && nx < cols && mtx[ny][nx] {
			res |= d.flag
		}
	}

	return res
}

// halftoneImage is an image.Gray type image, which At(x, y) return color.Gray.
// black equals to color.Gray{0}, white equals to color.Gray{255}.
func halftoneColor(halftoneImage image.Image, transparent bool, x, y int) color.Color {

	c0 := halftoneImage.At(x, y)
	c1, ok := halftoneImage.At(x, y).(color.Gray)
	if !ok {
		log.Printf("halftoneColor: not a gray image, got: %T\n", c0)
		return c0
	}

	if c1.Y == 255 {
		if transparent {
			return color.RGBA{}
		}
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}

	return color.RGBA{A: 255}
}

func validLogoImage(qrWidth, qrHeight, logoWidth, logoHeight, logoSizeMultiplier int) bool {
	return qrWidth >= logoSizeMultiplier*logoWidth && qrHeight >= logoSizeMultiplier*logoHeight
}

func blockOverlapsLogo(x, y, blockSize, left, top, w, h, logoWidth, logoHeight int) bool {
	blockLeft := x*blockSize + left
	blockTop := y*blockSize + top
	blockRight := blockLeft + blockSize
	blockBottom := blockTop + blockSize

	logoLeft := (w - logoWidth) / 2
	logoTop := (h - logoHeight) / 2
	logoRight := logoLeft + logoWidth
	logoBottom := logoTop + logoHeight

	return blockRight > logoLeft && blockLeft < logoRight &&
		blockBottom > logoTop && blockTop < logoBottom
}

// Attribute contains basic information of generated image.
type Attribute struct {
	// width and height of image
	W, H int
	// in the order of "top, right, bottom, left"
	Borders [4]int
	// the length of  block edges
	BlockWidth int
}
