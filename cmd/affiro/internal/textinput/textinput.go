package textinput

import (
	"image/color"
	"sync"
	"time"

	"github.com/oakmound/oak/v4/alg/floatgeom"
	"github.com/oakmound/oak/v4/entities"
	"github.com/oakmound/oak/v4/event"
	"github.com/oakmound/oak/v4/key"
	"github.com/oakmound/oak/v4/mouse"
	"github.com/oakmound/oak/v4/render"
	"github.com/oakmound/oak/v4/scene"
	"github.com/oakmound/oak/v4/timing"
)

// TextInput provides a nicer way to handle input of text
// Notably it creates a blinking input cursor
// Fields are grouped by what they belong to rather than packed by size: one TextInput is
// built per input, so the bytes are not worth scattering the locks away from what they guard.
type TextInput struct { //nolint:govet // fieldalignment
	*entities.Entity
	ctx *scene.Context

	parentCID event.CallerID

	bindingLock sync.Mutex

	textLock    sync.Mutex
	currentText *string

	editing bool
	x, y    float64
	w, h    float64

	// TODO:
	//nolint:unused
	position   floatgeom.Point2
	textOffset floatgeom.Point2

	finalizer   func(string)
	onFirstEdit func(ti *TextInput)
	onEdit      func(ti *TextInput)

	font *render.Font

	blinkerLock   sync.Mutex
	blinker       render.Renderable
	blinkerColor  color.Color
	blinkRate     time.Duration
	blinkerIndex  int
	blinkerLayers []int

	onClick, onDown, onHeld event.Binding

	sensitive     bool
	sensitiveText string

	// entityOptions []entities.Option
}

// New textinput for the scene given a set of options
func New(ctx *scene.Context, opts ...Option) *TextInput {
	emptyString := ""
	ti := &TextInput{
		ctx:           ctx,
		w:             100,
		h:             20,
		font:          render.DefaultFont(),
		blinkerColor:  color.RGBA{255, 255, 255, 255},
		currentText:   &emptyString,
		blinkerLayers: []int{0, 2},
	}
	for _, opt := range opts {
		opt(ti)
	}
	ti.font = ti.font.Copy()
	r := ti.font.NewStrPtrText(ti.currentText, 0, 0)

	ti.parentCID = ctx.Register(ti)

	ti.Entity = entities.New(ti.ctx,
		entities.WithPosition(floatgeom.Point2{ti.x, ti.y}),
		entities.WithDimensions(floatgeom.Point2{ti.w, ti.h}),
		entities.WithRenderable(r), entities.WithParent(ti),
		entities.WithUseMouseTree(true),
	)
	ti.bindStartTyping()
	ti.Renderable.SetPos(ti.x+ti.textOffset.X(), ti.y+ti.textOffset.Y())
	return ti
}

func (ti *TextInput) CID() event.CallerID {
	return ti.parentCID
}

func (ti *TextInput) bindStartTyping() {
	event.Bind(ti.ctx, mouse.RelativeClickOn, ti, func(ti *TextInput, me *mouse.Event) event.Response {
		return ti.startTyping(*me)
	})
}

// startTyping bind initiates the ability to add text to the textinput area
func (ti *TextInput) startTyping(me mouse.Event) event.Response {
	ti.bindingLock.Lock()
	defer ti.bindingLock.Unlock()

	if ti.onFirstEdit != nil {
		ti.onFirstEdit(ti)
		ti.onFirstEdit = nil
	}
	if ti.onEdit != nil {
		ti.onEdit(ti)
	}
	ti.editing = true
	ti.updateBlinkerToMouse(me)
	ti.onDown = event.Bind(ti.ctx, key.AnyDown, ti, editBinding)
	ti.onHeld = event.Bind(ti.ctx, key.AnyHeld, ti, editBinding)
	ti.onClick = event.Bind(ti.ctx, mouse.Click, ti, func(ti *TextInput, ev *mouse.Event) event.Response {
		return ti.stopTyping()
	})
	return event.ResponseUnbindThisBinding
}

func (ti *TextInput) stopTyping() event.Response {
	ti.bindingLock.Lock()
	defer ti.bindingLock.Unlock()
	// only stop editing if not already editing
	if !ti.editing {
		return event.ResponseUnbindThisBinding
	}

	ti.editing = false
	ti.undrawBlinker()
	if ti.finalizer != nil {
		if ti.sensitive {
			ti.finalizer(ti.sensitiveText)
		} else {
			ti.finalizer(*ti.currentText)
		}
	}
	ti.bindStartTyping()
	ti.onDown.Unbind()
	ti.onHeld.Unbind()
	return event.ResponseUnbindThisBinding
}

func (ti *TextInput) undrawBlinker() {
	ti.blinkerLock.Lock()
	defer ti.blinkerLock.Unlock()
	if ti.blinker != nil {
		ti.blinker.Undraw()
	}
}

func editBinding(ti *TextInput, k key.Event) event.Response {
	// safety check that we are actually editing
	if !ti.editing {
		return event.ResponseUnbindThisBinding
	}

	ti.textLock.Lock()
	txt := *ti.currentText
	ti.textLock.Unlock()

	shift := 0

	switch k.Code {
	case key.ReturnEnter, key.Escape:
		return ti.finishEditing(txt)
	case key.DeleteBackspace:
		txt = ti.deleteBack(txt)
		shift = -1
	case key.LeftShift, key.RightShift, key.Tab:
	case key.LeftArrow:
		ti.updateBlinkerRelative(-1)
		return 0
	case key.RightArrow:
		ti.updateBlinkerRelative(1)
		return 0
	default:
		txt = ti.insertRune(txt, k.Rune)
		shift = len(string(k.Rune))
	}
	ti.textLock.Lock()
	*ti.currentText = txt
	ti.textLock.Unlock()
	ti.updateBlinkerRelative(shift)

	return 0
}

// finishEditing hands the finished text to the finalizer and puts the input back into its
// waiting-to-be-clicked state.
func (ti *TextInput) finishEditing(txt string) event.Response {
	ti.bindingLock.Lock()
	defer ti.bindingLock.Unlock()
	ti.editing = false
	ti.undrawBlinker()
	if ti.finalizer != nil {
		if ti.sensitive {
			ti.finalizer(ti.sensitiveText)
		} else {
			ti.finalizer(txt)
		}
	}
	ti.bindStartTyping()
	return event.ResponseUnbindThisBinding
}

// deleteBack removes the character before the blinker, from the sensitive text too when the
// input is masked.
func (ti *TextInput) deleteBack(txt string) string {
	txt = removeAt(txt, ti.blinkerIndex)
	if ti.sensitive {
		ti.sensitiveText = removeAt(ti.sensitiveText, ti.blinkerIndex)
	}
	return txt
}

// removeAt drops the character before index i, leaving s alone when there is nothing there.
func removeAt(s string, i int) string {
	if len(s) == 0 || i == 0 {
		return s
	}
	if i >= len(s) {
		return s[:i-1]
	}
	return s[:i-1] + s[i:]
}

// insertRune places r at the blinker. A masked input shows a star and keeps the real
// character aside; a NUL rune is not a character at all and is dropped.
func (ti *TextInput) insertRune(txt string, r rune) string {
	switch {
	case ti.sensitive:
		ti.sensitiveText = ti.sensitiveText[:ti.blinkerIndex] + string(r) + ti.sensitiveText[ti.blinkerIndex:]
		return txt + "*"
	case string(r) == "\x00":
		return txt
	default:
		return txt[:ti.blinkerIndex] + string(r) + txt[ti.blinkerIndex:]
	}
}

// blinker for showing where you are performing inputs

// updateBlinkerToMouse sets the blinker to roughly wheref the mouse was clicking.
// Allows for setting at a reasonable space within the given text
func (ti *TextInput) updateBlinkerToMouse(me mouse.Event) {
	ti.textLock.Lock()
	// convert me to index position
	// linear scan until its demonstrated we need something with better performance
	var textIndex int
	for i := range len(*ti.currentText) {
		charX := float64(ti.font.MeasureString((*ti.currentText)[:i]).Round())
		charX += ti.Renderable.X()
		if charX > me.X() {
			textIndex = i
			break
		}
	}
	ti.textLock.Unlock()

	ti.updateBlinker(textIndex)
}

func (ti *TextInput) updateBlinkerRelative(shift int) {
	ti.updateBlinker(ti.blinkerIndex + shift)
}

func (ti *TextInput) updateBlinker(textIndex int) {
	ti.blinkerLock.Lock()
	defer ti.blinkerLock.Unlock()
	if ti.blinker != nil {
		ti.blinker.Undraw()
	}
	ti.textLock.Lock()
	var w float64
	h := ti.font.Height()
	if textIndex < 0 {
		w = 0
		ti.blinkerIndex = 0
	} else {
		if textIndex >= len(*ti.currentText) {
			textIndex = len(*ti.currentText)
		}
		fixedWidth := ti.font.MeasureString((*ti.currentText)[:textIndex])
		w = float64(fixedWidth.Round())
		ti.blinkerIndex = textIndex
	}
	x := ti.X()
	y := ti.Y()
	ti.textLock.Unlock()
	if ti.blinkRate != 0 {
		ti.blinker = render.NewSequence(timing.FrameDelayToFPS(ti.blinkRate),
			render.NewLine(x+w, y, x+w, y+h, ti.blinkerColor),
			render.EmptyRenderable(),
		)
	} else {
		ti.blinker = render.NewLine(x+w, y, x+w, y+h, ti.blinkerColor)
	}
	//nolint:errcheck
	ti.ctx.Draw(ti.blinker, ti.blinkerLayers...)
}

// Select the textinput for cases where you need to simulate mouse clicks
func (ti *TextInput) Select() {
	event.TriggerForCallerOn(ti.ctx, ti.CallerID, mouse.ClickOn, &mouse.Event{})
}

// Deselect the textinput for cases where you need to simulate mouse clicks
func (ti *TextInput) Deselect() {
	ti.stopTyping()
}
