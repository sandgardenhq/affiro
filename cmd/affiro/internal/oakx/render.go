package oakx

import (
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"math"

	"github.com/fyne-io/oksvg"
	"github.com/oakmound/oak/v4/alg/floatgeom"
	"github.com/oakmound/oak/v4/entities"
	"github.com/oakmound/oak/v4/event"
	"github.com/oakmound/oak/v4/mouse"
	"github.com/oakmound/oak/v4/render"
	"github.com/srwiley/rasterx"
)

func NewFilledCircle(c color.Color, radius, thickness float64, offsets ...float64) *render.Sprite {
	sp := render.NewEmptySprite(0, 0, int(radius)*2, int(radius)*2)
	render.DrawCircle(sp.GetRGBA(), c, radius, thickness, offsets...)
	offX := 0.0
	offY := 0.0
	if len(offsets) > 0 {
		offX = offsets[0]
		if len(offsets) > 1 {
			offY = offsets[1]
		}
	}
	rVec := floatgeom.Point2{radius + offX, radius + offY}
	for x := range int(radius * 2) {
		for y := range int(radius * 2) {
			candidate := floatgeom.Point2{float64(x), float64(y)}
			distance := math.Abs(rVec.Distance(candidate))
			if distance <= radius {
				sp.Set(x, y, c)
			}
		}
	}
	return sp
}

// This is notably smoother than render.DrawCircle
func NewCircleAlt(x, y, r int, c color.Color) *render.Sprite {
	sp := render.NewEmptySprite(0, 0, int(r)*2+1, int(r)*2+1)
	rgba := sp.GetRGBA()
	if r < 0 {
		return sp
	}
	cx := x + r
	cy := y + r
	x = cx
	y = cy
	// Bresenham algorithm
	x1, y1, err := -r, 0, 2-2*r
	for {
		rgba.Set(x-x1, y+y1, c)
		rgba.Set(x-y1, y-x1, c)
		rgba.Set(x+x1, y-y1, c)
		rgba.Set(x+y1, y+x1, c)
		r = err
		if r > x1 {
			x1++
			err += x1*2 + 1
		}
		if r <= y1 {
			y1++
			err += y1*2 + 1
		}
		if x1 >= 0 {
			break
		}
	}
	// fill
	for x2 := 0; x2 < (r*2)+1; x2++ {
		drawingCol := false
		for y2 := 0; y2 < (r*2)+1; y2++ {
			if !drawingCol {
				// hit the top of the outline
				if rgba.RGBAAt(x2, y2) == c {
					drawingCol = true
				}
			} else {
				// hit the bottom of the outline
				if y2 > cy && rgba.RGBAAt(x2, y2) == c {
					break
				}
				rgba.Set(x2, y2, c)
			}
		}
	}
	return sp
}

func InvertColors(rgba *image.RGBA) {
	bounds := rgba.Bounds()
	w := bounds.Max.X
	h := bounds.Max.Y
	for x := range w {
		for y := range h {
			r, g, b, a := rgba.At(x, y).RGBA()
			midPoint := uint32(0xffff >> 1)
			fmt.Print("in: ", r, g, b, a, midPoint, "  ")
			if r < midPoint {
				r += 2 * (midPoint - r)
			} else {
				r -= 2 * (r - midPoint)
			}
			if g < midPoint {
				g += 2 * (midPoint - g)
			} else {
				g -= 2 * (g - midPoint)
			}
			if b < midPoint {
				b += 2 * (midPoint - b)
			} else {
				b -= 2 * (b - midPoint)
			}
			// a is unchanged
			newRGBA := color.RGBA64{
				uint16(r),
				uint16(g),
				uint16(b),
				uint16(a),
			}
			fmt.Println("out: ", r, g, b, a)
			rgba.Set(x, y, newRGBA)
		}
	}
}

func LoadSVG(fs fs.FS, path string, width, height int, defaultColor string) *render.Sprite {
	svgFile, err := fs.Open(path)
	if err != nil {
		panic(err)
	}
	//nolint:errcheck
	defer svgFile.Close()
	icon, err := oksvg.ReadReplacingCurrentColor(svgFile, defaultColor)
	if err != nil {
		panic(err)
	}
	inputW, inputH := icon.ViewBox.W, icon.ViewBox.H
	iconAspect := inputW / inputH
	buff := image.NewRGBA(image.Rect(0, 0, width, height))

	viewAspect := float64(width) / float64(height)
	outputW, outputH := width, height
	if viewAspect > iconAspect {
		outputW = int(float64(width) * iconAspect)
	} else if viewAspect < iconAspect {
		outputH = int(float64(height) / iconAspect)
	}
	scanner := rasterx.NewScannerGV(int(inputW), int(inputH), buff, image.Rect(0, 0, width, height))
	scanner.SetBounds(10000, 10000)
	dasher := rasterx.NewDasher(width, height, scanner)
	icon.SetTarget(0, 0, float64(outputW), float64(outputH))
	icon.Draw(dasher, 1)
	return render.NewSprite(0, 0, buff)
}

func EntitySwitchBinding(k string) func(b *entities.Entity, _ *mouse.Event) event.Response {
	return func(b *entities.Entity, _ *mouse.Event) event.Response {
		s, ok := b.Renderable.(*render.Switch)
		if ok {
			//nolint:errcheck
			s.Set(k)
		}
		return 0
	}
}
