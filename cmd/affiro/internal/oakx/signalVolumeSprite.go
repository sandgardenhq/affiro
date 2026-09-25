package oakx

import (
	"image/color"

	"github.com/oakmound/oak/v4/render"
)

type SignalVolumeSprite struct {
	barColors [5]color.Color
	*render.Sprite
	thresholds   [4]int
	totalActions int
}

func NewSignalVolumeSprite(x, y float64, w, h int) *SignalVolumeSprite {
	sp := render.NewEmptySprite(x, y, w, h)
	return &SignalVolumeSprite{
		Sprite: sp,
		thresholds: [4]int{
			5,
			50,
			300,
			1500,
		},
		barColors: [5]color.Color{
			0: color.RGBA{128, 0, 0, 255},
			1: color.RGBA{128, 128, 0, 255},
			2: color.RGBA{0, 128, 0, 255},
			3: color.RGBA{0, 200, 0, 255},
			4: color.RGBA{0, 255, 0, 255},
		},
	}
}

func (s *SignalVolumeSprite) SetActions(totalActions int) {
	s.totalActions = totalActions
	if s.totalActions == 0 {
		return
	}
	bars := 1
	for _, v := range s.thresholds {
		if totalActions < v {
			break
		}
		bars++
	}
	color := s.barColors[bars-1]
	bds := s.GetRGBA().Bounds()
	barXDistance := bds.Max.X / 6
	barHeight := bds.Max.Y / 5
	x := 1
	for b := range bars {
		render.DrawLine(s.GetRGBA(), x, bds.Max.Y, x, bds.Max.Y-(barHeight*b), color)
		x += barXDistance
	}
}
