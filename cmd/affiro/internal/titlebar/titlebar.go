package titlebar

import (
	"image"
	"image/color"
	"slices"
	"strconv"
	"time"

	oak "github.com/oakmound/oak/v4"
	"github.com/oakmound/oak/v4/alg/floatgeom"
	"github.com/oakmound/oak/v4/alg/intgeom"
	"github.com/oakmound/oak/v4/entities"
	"github.com/oakmound/oak/v4/entities/x/btn"
	"github.com/oakmound/oak/v4/event"
	"github.com/oakmound/oak/v4/mouse"
	"github.com/oakmound/oak/v4/render"
	"github.com/oakmound/oak/v4/render/mod"
	"github.com/oakmound/oak/v4/scene"
	"github.com/oakmound/oak/v4/shape"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/oakx"
)

// revertSuffix marks the states a maximize button shows once the window is maximized, where
// the icon and the click both revert to the normal size. toggleMaximize stores it as the
// button's switch-suffix so a hover or press can rebuild the key it belongs on.
const revertSuffix = "-revert"

// The titlebar keys its switches by the shared button states, plus a reverted variant of each
// for the maximize button.
const (
	stateUnhoverRevert = oakx.StateUnhover + revertSuffix
	stateHoverRevert   = oakx.StateHover + revertSuffix
	statePressRevert   = oakx.StatePress + revertSuffix
)

type TitleBar struct {
	lastPressAt        time.Time
	buttons            map[Button]*entities.Entity
	draggingStartPos   floatgeom.Point2
	startingDimensions intgeom.Point2
	draggingWindow     bool
	maximized          bool
}

// Fields are grouped by what they configure rather than packed by size; one Constructor
// describes one window.
type Constructor struct { //nolint:govet // fieldalignment
	Color          color.Color
	HighlightColor color.Color
	MouseDownColor color.Color
	Height         float64
	Layers         []int

	Title          string
	TitleFontSize  int
	TitleXOffset   int
	TitleTextColor color.Color

	Buttons              []Button
	ButtonWidth          float64
	ButtonStyle          ButtonStyle
	DoubleClickThreshold time.Duration
}

type ButtonStyle int

const (
	ButtonStyleDefault ButtonStyle = 0
	ButtonStyleOSX     ButtonStyle = 1
)

type Button uint8

// Buttons to show on the title bar
const (
	ButtonClose    Button = iota
	ButtonMinimize Button = iota
	ButtonMaximize Button = iota
)

var DefaultConstructor = Constructor{
	Color:  color.RGBA{128, 128, 128, 255},
	Height: 32,
	Layers: []int{},
	Buttons: []Button{
		ButtonMinimize,
		ButtonMaximize,
		ButtonClose,
	},
	ButtonWidth:          32,
	TitleFontSize:        17,
	TitleXOffset:         10,
	TitleTextColor:       color.RGBA{255, 255, 255, 255},
	DoubleClickThreshold: 200 * time.Millisecond,
}

var WindowClosingEvent = event.RegisterEvent[struct{}]()

// New constructs a new TitleBar
func New(ctx *scene.Context, opts ...Option) *TitleBar {
	construct := DefaultConstructor
	for _, opt := range opts {
		construct = opt(construct)
	}
	if construct.HighlightColor == nil {
		construct.HighlightColor = mod.Lighter(construct.Color, .10)
	}
	if construct.MouseDownColor == nil {
		construct.MouseDownColor = mod.Lighter(construct.Color, .20)
	}

	screenBounds := ctx.Window.Bounds()
	screenHeight := screenBounds.Y()
	screenWidth := screenBounds.X()

	font, _ := render.DefaultFont().RegenerateWith(func(fg render.FontGenerator) render.FontGenerator {
		fg.Size = float64(construct.TitleFontSize)
		fg.Color = image.NewUniform(construct.TitleTextColor)
		return fg
	})

	dragBarWidth := float64(screenWidth)

	totalButtonsSize := construct.ButtonWidth * float64(len(construct.Buttons))
	dragBarWidth -= totalButtonsSize

	hdr := &TitleBar{
		lastPressAt:        time.Now(),
		buttons:            make(map[Button]*entities.Entity),
		startingDimensions: intgeom.Point2{screenWidth, screenHeight},
	}
	if construct.ButtonStyle == ButtonStyleOSX {
		// sort close, min, max
		slices.Sort(construct.Buttons)
	}
	hdr.addButtons(ctx, construct, dragBarWidth)
	hdr.addDragBar(ctx, construct, font, dragBarWidth, screenWidth)
	return hdr
}

// addButtons builds each configured button and places it along the right of the bar, or
// along the left when the OSX style is asked for.
func (hdr *TitleBar) addButtons(ctx *scene.Context, construct Constructor, dragBarWidth float64) {
	for i, button := range construct.Buttons {
		r, txt, btnOffset, clickBinding := hdr.buildButton(ctx, construct, button, i)
		bw := construct.ButtonWidth
		bh := construct.Height
		x := btnOffset.X() + float64(i)*construct.ButtonWidth
		y := btnOffset.Y()
		if construct.ButtonStyle != ButtonStyleOSX {
			x += dragBarWidth
		} else {
			bw /= 2
			x -= float64(i) * bw
			bh /= 2
		}
		hdr.buttons[button] = btn.New(ctx,
			btn.Text(txt),
			btn.Pos(x, y),
			btn.Renderable(r),
			btn.Height(bh),
			btn.Width(bw),
			btn.Layers(construct.Layers...),
			btn.Binding(mouse.Start, setSwitchState(oakx.StateHover)),
			btn.Binding(mouse.Stop, unhover),
			btn.Binding(mouse.PressOn, setSwitchState(oakx.StatePress)),
			btn.Click(clickBinding),
			btn.Binding(oak.FocusLoss, func(b *entities.Entity, _ struct{}) event.Response {
				if construct.ButtonStyle == ButtonStyleOSX {
					if sw, ok := b.Renderable.(*render.Switch); ok {
						//nolint:errcheck
						sw.Set("nofocus")
					}
					b.SetMetadata("nofocus", "on")
				}
				return 0
			}),
			btn.Binding(oak.FocusGain, func(b *entities.Entity, _ struct{}) event.Response {
				if construct.ButtonStyle == ButtonStyleOSX {
					if sw, ok := b.Renderable.(*render.Switch); ok {
						suffix, _ := b.Metadata("switch-suffix")
						//nolint:errcheck
						sw.Set(oakx.StateUnhover + suffix)
					}
					b.SetMetadata("nofocus", "")
				}
				return 0
			}),
			// btn.Binding(oak.WindowSizeChange, oak.SizeChangeEvent(func(c event.CID, pt intgeom.Point2) int {
			// 	b, _ := ctx.CallerMap.GetEntity(c).(*btn.Box)
			// 	b.SetPos(float64(pt.X())-totalButtonsSize+float64(i)*construct.ButtonWidth, 0)
			// 	return 0
			// })),
		)
	}
}

// dragStep moves the window by however far the cursor has travelled since the drag started.
// lastDragPos guards against acting twice on one position, which has only been seen on
// linux/x11 and may not be needed elsewhere.
func (hdr *TitleBar) dragStep(ctx *scene.Context, lastDragPos *floatgeom.Point2) {
	x, y := ctx.Window.(*oak.Window).GetCursorPosition()
	pt := floatgeom.Point2{x, y}
	delta := pt.Sub(hdr.draggingStartPos)
	if delta == (floatgeom.Point2{}) || pt == *lastDragPos {
		return
	}
	newX, newY := ctx.Window.GetDesktopPosition()
	newX += delta.X()
	newY += delta.Y()
	if hdr.maximized {
		if mxbtn, ok := hdr.buttons[ButtonMaximize]; ok {
			hdr.maximized = toggleMaximize(ctx, mxbtn)
		}
	}
	screenBounds := ctx.Window.Bounds()
	screenHeight := screenBounds.Y()
	screenWidth := screenBounds.X()
	scale := ctx.Window.Scale()
	//nolint:errcheck
	ctx.Window.MoveWindow(int(newX), int(newY), int(float64(screenWidth)*scale), int(float64(screenHeight)*scale))
	if !floatgeom.NewRect2WH(0, 0, float64(screenWidth), float64(screenHeight)).Contains(hdr.draggingStartPos) {
		hdr.draggingStartPos = floatgeom.Point2{
			float64(screenWidth) / 2, 16,
		}
	}
	*lastDragPos = pt
}

// setSwitchState returns a binding that moves a button's render.Switch to the named state,
// keeping whatever suffix the button carries (the maximize button has a reverted set).
func setSwitchState(state string) func(*entities.Entity, *mouse.Event) event.Response {
	return func(b *entities.Entity, _ *mouse.Event) event.Response {
		if sw, ok := b.Renderable.(*render.Switch); ok {
			suffix, _ := b.Metadata("switch-suffix")
			//nolint:errcheck
			sw.Set(state + suffix)
		}
		return 0
	}
}

// unhover returns a button to rest, or to its unfocused look when the window has lost focus.
func unhover(b *entities.Entity, _ *mouse.Event) event.Response {
	sw, ok := b.Renderable.(*render.Switch)
	if !ok {
		return 0
	}
	if nofocus, _ := b.Metadata("nofocus"); nofocus == "on" {
		//nolint:errcheck
		sw.Set("nofocus")
		return 0
	}
	suffix, _ := b.Metadata("switch-suffix")
	//nolint:errcheck
	sw.Set(oakx.StateUnhover + suffix)
	return 0
}

// buildButton produces the renderable, label, offset and click handler for one titlebar
// button. A button this bar does not know about keeps its index as its label.
func (hdr *TitleBar) buildButton(ctx *scene.Context, construct Constructor, button Button, i int) (render.Modifiable, string, floatgeom.Point2, func(*entities.Entity, *mouse.Event) event.Response) {
	var r render.Modifiable = render.NewColorBox(int(construct.ButtonWidth), int(construct.Height), construct.Color)
	txt := strconv.Itoa(i)
	var clickBinding = func(*entities.Entity, *mouse.Event) event.Response {
		return 0
	}
	var btnOffset floatgeom.Point2

	switch button {
	case ButtonMinimize:
		if construct.ButtonStyle == ButtonStyleDefault {
			r = render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
				oakx.StateUnhover: SpriteFromShape(minimizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.Color),
				oakx.StateHover:   SpriteFromShape(minimizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.HighlightColor),
				oakx.StatePress:   SpriteFromShape(minimizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.MouseDownColor),
			})
		} else {
			// TODO: this duplicates, poorly, native osx buttons; we ideally could reuse them and put our content in the same area as the native top bar;
			// this requires more direct interfacing with the NSWindow from objc
			nohover := oakx.NewCircleAlt(0, 0, int(construct.ButtonWidth/6), color.RGBA{255, 0xcc, 0, 255})
			nofocus := oakx.NewCircleAlt(0, 0, int(construct.ButtonWidth/6), color.RGBA{0xaa, 0xaa, 0xaa, 255})
			press := oakx.NewCircleAlt(0, 0, int(construct.ButtonWidth/6), color.RGBA{0xaa, 0x99, 0, 255})
			icon := SpriteFromShape(thickMinimizeIcon, int(construct.ButtonWidth*(2.0/5)), int(construct.Height*(2.0/5)), color.RGBA{100, 100, 100, 255}, color.RGBA{0, 0, 0, 0})
			icon.SetPos(-3, -3)
			r = render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
				oakx.StateUnhover: nohover,
				"nofocus":         nofocus,
				oakx.StateHover:   render.NewCompositeM(nohover, icon),
				oakx.StatePress:   render.NewCompositeM(press, icon),
			})
			btnOffset = floatgeom.Point2{construct.ButtonWidth / 4, construct.Height / 3}
		}
		txt = ""
		clickBinding = func(*entities.Entity, *mouse.Event) event.Response {
			//nolint:errcheck
			ctx.Window.(*oak.Window).Minimize()
			return 0
		}
	case ButtonClose:
		if construct.ButtonStyle == ButtonStyleDefault {
			r = render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
				oakx.StateUnhover: SpriteFromShape(closeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.Color),
				oakx.StateHover:   SpriteFromShape(closeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.HighlightColor),
				oakx.StatePress:   SpriteFromShape(closeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.MouseDownColor),
			})
		} else {
			nohover := oakx.NewCircleAlt(0, 0, int(construct.ButtonWidth/6), color.RGBA{255, 0, 0, 255})
			nofocus := oakx.NewCircleAlt(0, 0, int(construct.ButtonWidth/6), color.RGBA{0xaa, 0xaa, 0xaa, 255})
			press := oakx.NewCircleAlt(0, 0, int(construct.ButtonWidth/6), color.RGBA{0xcc, 0, 0, 255})
			icon := SpriteFromShape(thickCloseIcon, int(construct.ButtonWidth*(2.0/5)), int(construct.Height*(2.0/5)), color.RGBA{30, 30, 30, 255}, color.RGBA{0, 0, 0, 0})
			icon.SetPos(-3, -3)
			r = render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
				oakx.StateUnhover: nohover,
				"nofocus":         nofocus,
				oakx.StateHover:   render.NewCompositeM(nohover, icon),
				oakx.StatePress:   render.NewCompositeM(press, icon),
			})
			btnOffset = floatgeom.Point2{construct.ButtonWidth / 4, construct.Height / 3}
		}
		txt = ""
		clickBinding = func(*entities.Entity, *mouse.Event) event.Response {
			<-event.TriggerOn(ctx, WindowClosingEvent, struct{}{})
			ctx.Window.Quit()
			return 0
		}
	case ButtonMaximize:
		if construct.ButtonStyle == ButtonStyleDefault {
			r = render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
				oakx.StateUnhover:  SpriteFromShape(maximizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.Color),
				oakx.StateHover:    SpriteFromShape(maximizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.HighlightColor),
				oakx.StatePress:    SpriteFromShape(maximizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.MouseDownColor),
				stateUnhoverRevert: SpriteFromShape(normalizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.Color),
				stateHoverRevert:   SpriteFromShape(normalizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.HighlightColor),
				statePressRevert:   SpriteFromShape(normalizeIcon, int(construct.ButtonWidth), int(construct.Height), color.RGBA{255, 255, 255, 255}, construct.MouseDownColor),
			})
		} // else: TODO: OSX maximize button
		txt = ""

		clickBinding = func(b *entities.Entity, _ *mouse.Event) event.Response {
			hdr.maximized = toggleMaximize(ctx, b)
			return 0
		}
	}
	return r, txt, btnOffset, clickBinding
}

// addDragBar builds the title area, which doubles as the handle the window is dragged by and
// double-clicked to maximise.
func (hdr *TitleBar) addDragBar(ctx *scene.Context, construct Constructor, font *render.Font, dragBarWidth float64, screenWidth int) {
	layers := construct.Layers
	if len(layers) != 0 {
		layers[len(layers)-1]--
	}
	var lastDragPos floatgeom.Point2
	btn.New(ctx,
		btn.Font(font),
		btn.Text(construct.Title),
		btn.TxtOff(10, construct.Height/2-float64(construct.TitleFontSize)/2),
		btn.Layers(layers...),
		btn.Width(dragBarWidth),
		btn.Height(construct.Height),
		btn.Color(construct.Color),
		btn.Binding(mouse.PressOn, func(_ *entities.Entity, ev *mouse.Event) event.Response {
			if time.Since(hdr.lastPressAt) < construct.DoubleClickThreshold {
				if mxbtn, ok := hdr.buttons[ButtonMaximize]; ok {
					hdr.maximized = toggleMaximize(ctx, mxbtn)
				}
				// if this is not set, dragging can persist after the window shrinks
				hdr.draggingWindow = false
				return 0
			}
			hdr.lastPressAt = time.Now()
			hdr.draggingWindow = true
			x, y := ctx.Window.(*oak.Window).GetCursorPosition()
			hdr.draggingStartPos = floatgeom.Point2{float64(x), float64(y)}
			return 0
		}),
		// Q: Why not mouse.Drag?
		// A: mouse.Drag is only triggered for on-screen mouse events. If the mouse
		//    falls out of the window, as it likely will if you drag the window up,
		//    the window will freeze until you bring the mouse cursor back into the window.
		btn.Binding(event.Enter, func(_ *entities.Entity, ev event.EnterPayload) event.Response {
			if hdr.draggingWindow {
				hdr.dragStep(ctx, &lastDragPos)
			}
			return 0
		}),
		btn.Binding(mouse.Release, func(_ *entities.Entity, ev *mouse.Event) event.Response {
			if hdr.draggingWindow {
				hdr.draggingWindow = false
			}
			return 0
		}),
		// btn.Binding(oak.WindowSizeChange, oak.SizeChangeEvent(func(c event.CID, pt intgeom.Point2) int {
		// 	ctx.Window.(*oak.Window).UpdateViewSize(pt.X(), pt.Y())
		// 	b, _ := ctx.CallerMap.GetEntity(c).(*btn.TextBox)
		// 	newW := float64(pt.X()) - totalButtonsSize
		// 	b.Box.R.Undraw()
		// 	b.Box.R = render.NewColorBox(int(newW), int(construct.Height), construct.Color)
		// 	ctx.DrawStack.Draw(b.Box.R, construct.Layers...)
		// 	ctx.MouseTree.UpdateSpace(0, 0, newW, construct.Height, b.Box.Space)
		// 	return 0
		// })),
	)
	btn.New(ctx,
		btn.Layers(layers...),
		btn.Width(float64(screenWidth-int(dragBarWidth))),
		btn.Pos(dragBarWidth, 0),
		btn.Height(construct.Height),
		btn.Color(construct.Color),
	)
}

var closeIcon = shape.JustIn(shape.AndIn(
	shape.XRange(.35, .65),
	func(x, y int, sizes ...int) bool {
		size := sizes[0]
		return x == y || y == (size-x)
	},
))

var thickCloseIcon = shape.JustIn(shape.AndIn(
	shape.XRange(.35, .65),
	func(x, y int, sizes ...int) bool {
		size := sizes[0]
		xyDelta := x - y
		sizeDelta := y - (size - x)
		if xyDelta > -2 && xyDelta < 2 {
			return true
		}
		if sizeDelta > -2 && sizeDelta < 2 {
			return true
		}
		return false
	},
))

var minimizeIcon = shape.JustIn(shape.AndIn(
	shape.XRange(.35, .65),
	func(x, y int, sizes ...int) bool {
		return y == sizes[0]/2
	},
))

var thickMinimizeIcon = shape.JustIn(shape.AndIn(
	shape.XRange(.35, .65),
	func(x, y int, sizes ...int) bool {
		delta := y - sizes[0]/2
		return delta > -2 && delta < 2
	},
))

var maximizeIcon = shape.JustIn(squarePercent(.35, .65))

var normalizeIcon = shape.JustIn(shape.OrIn(
	squarePercent(.35, .65),
	squarePercent(.45, .55),
))

func squarePercent(minPerc, maxPerc float64) shape.In {
	return shape.AndIn(
		shape.XRange(minPerc-.03, maxPerc),
		func(x, y int, sizes ...int) bool {
			yf := float64(y)
			sf := float64(sizes[0])
			return (yf >= sf*(minPerc-.03)) && (yf <= sf*maxPerc)
		},
		func(x, y int, sizes ...int) bool {
			size := sizes[0]
			return x == int(float64(size)*minPerc) ||
				x == int(float64(size)*maxPerc) ||
				y == int(float64(size)*minPerc) ||
				y == int(float64(size)*maxPerc)
		},
	)
}

func toggleMaximize(ctx *scene.Context, b *entities.Entity) bool {
	if sfx, _ := b.Metadata("switch-suffix"); sfx != "" {
		//nolint:errcheck
		ctx.Window.(*oak.Window).SetFullScreen(false)
		b.SetMetadata("switch-suffix", "")
		if sw, ok := b.Renderable.(*render.Switch); ok {
			//nolint:errcheck
			sw.Set(oakx.StateUnhover)
		}
		return false
	}
	//nolint:errcheck
	ctx.Window.SetFullScreen(true)
	b.SetMetadata("switch-suffix", revertSuffix)
	if sw, ok := b.Renderable.(*render.Switch); ok {
		//nolint:errcheck
		sw.Set(stateUnhoverRevert)
	}
	return true
}

func SpriteFromShape(sh shape.Shape, w, h int, on, off color.Color) *render.Sprite {
	rect := sh.Rect(w, h)
	rgba := image.NewRGBA(image.Rect(0, 0, len(rect), len(rect[0])))
	sp := render.NewSprite(0, 0, rgba)
	for x := range rect {
		for y := range len(rect[0]) {
			if rect[x][y] {
				sp.Set(x, y, on)
			} else {
				sp.Set(x, y, off)
			}
		}
	}
	return sp
}
