package main

import (
	"context"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/oakmound/oak/v4"
	"github.com/oakmound/oak/v4/alg/floatgeom"
	"github.com/oakmound/oak/v4/alg/intgeom"
	"github.com/oakmound/oak/v4/entities"
	"github.com/oakmound/oak/v4/entities/x/btn"
	"github.com/oakmound/oak/v4/event"
	okey "github.com/oakmound/oak/v4/key"
	"github.com/oakmound/oak/v4/mouse"
	"github.com/oakmound/oak/v4/render"
	"github.com/oakmound/oak/v4/render/mod"
	"github.com/oakmound/oak/v4/scene"
	"github.com/pkg/browser"
	"github.com/sandgardenhq/affiro/asig"
	"github.com/sandgardenhq/affiro/asig/keylog"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/asigx"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/auth"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/colors"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/keylogx"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/oakx"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/state"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/stringers"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/textinput"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/titlebar"
	"github.com/sandgardenhq/affiro/internal/buildinfo"
	"golang.org/x/mobile/event/key"

	_ "embed"
)

//go:embed images
var imagesFS embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := internal.ParseFlags()
	if err != nil {
		return err
	}
	if cfg.BackgroundWorker {
		return startBackgroundOnly(cfg.StorageDir)
	}
	fmt.Println(`backslash ('\') to copy`)
	st, err := state.New(context.Background(), cfg.StorageDir, time.Hour, cfg.APIURL)
	if err != nil {
		return fmt.Errorf("failed to open signature store: %w", err)
	}
	startBackground(st, cfg.StorageDir)
	if cfg.GUIMode {
		return guiMonitor(st, cfg)
	}
	if !keylogx.BackgroundWorkerAllowed {
		mon := keylog.NewMonitor()
		internal.GlobalMonitor.Set(mon)
		if err := mon.Start(); err != nil {
			return fmt.Errorf("failed to start keyboard monitor: %w", err)
		}
	}
	select {}
}

const reservedBackgroundFilePath = ".background-running"

func spawnBackgroundWorker(dir string) {
	// TODO: log these, and add a verbose flag to print these logs
	// fmt.Println("spawning background worker")
	knownPath := filepath.Join(dir, reservedBackgroundFilePath)
	spawned := false
	spawn := func() {
		if spawned {
			return
		}
		if err := keylogx.SpawnBackgroundWorker(); err != nil {
			panic(err)
		}
		spawned = true
		for {
			time.Sleep(1 * time.Second)
			f, err := os.Open(knownPath)
			if err == nil {
				fmt.Println("file found")
				f.Close()
				return
			}
			// fmt.Println("opening file error", err)
		}
	}
	for {
		f, err := os.Open(knownPath)
		if err != nil {
			// fmt.Println("file not found, spawning")
			spawn()
			continue
		}
		content, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}
		secs, err := strconv.ParseInt(string(content), 10, 64)
		if err != nil {
			// fmt.Println("invalid content found in background running file")
			spawn()
			continue
		}
		t := time.Unix(secs, 0)
		if time.Since(t) > 61*time.Second {
			// fmt.Println("background running file is too old!")
			spawn()
			continue
		}
		// fmt.Println("recent background running file found")
		return
	}
}

func recordEventsBackgroundWorker(st *state.State, dir string) {
	spawnBackgroundWorker(dir)
	ch, err := keylogx.WaitForBackgroundEvents()
	if err != nil {
		panic(err)
	}
	for e := range ch {
		// fmt.Println("pop:", e)
		if err := st.Write(e); err != nil {
			fmt.Println("error writing event: " + err.Error())
		}
		sigStr := st.String()
		if v, ok := e.(asig.KeyDownEvent); ok && v.Key == key.CodeBackslash {
			if err := clipboard.WriteAll(sigStr); err != nil {
				fmt.Println("error writing to clipboard: " + err.Error())
			}
		}
		fmt.Print(sigStr + "\r")
	}
}

func recordEventsForeground(st *state.State) {
	for {
		e, ok := internal.GlobalMonitor.Pop()
		// fmt.Println("pop:", e, ok)
		if !ok {
			// This is mostly just a problem on windows, as we need a better way to filter out OS events
			// we don't care about (or pop needs to loop though them)
			const inputRefreshRate = 20 * time.Millisecond // TODO: make user configurable
			time.Sleep(inputRefreshRate)
			continue
		}
		if err := st.Write(e); err != nil {
			fmt.Println("error writing event: " + err.Error())
		}
		sigStr := st.String()
		if v, ok := e.(asig.KeyDownEvent); ok && v.Key == key.CodeBackslash {
			if err := clipboard.WriteAll(sigStr); err != nil {
				fmt.Println("error writing to clipboard: " + err.Error())
			}
		}
		fmt.Print(sigStr + "\r")
	}
}

// recordEvents drains the key monitor into the signature for as long as the process runs,
// reprinting the signature after each event and copying it on backslash.
func recordEvents(st *state.State, dir string) {
	if keylogx.BackgroundWorkerAllowed {
		recordEventsBackgroundWorker(st, dir)
	} else {
		recordEventsForeground(st)
	}
}

// startBackground kicks off the work that runs for as long as the process does: flushing the
// signature to disk, draining key events into it, and stopping the monitor on a signal.
func startBackground(st *state.State, dir string) {
	const storageFlushRate = 30 * time.Second // TODO: make user configurable
	go func() {
		for range time.After(storageFlushRate) {
			if err := st.Refresh(); err != nil {
				fmt.Println(err)
			}
		}
	}()
	go recordEvents(st, dir)
	c := make(chan os.Signal, 10)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		internal.GlobalMonitor.Stop()
		os.Exit(1)
	}()
}

func keepBackgroundFileAlive(dir string) {
	check := func() {
		fmt.Println("keeping background file alive still")
		knownPath := filepath.Join(dir, reservedBackgroundFilePath)
		f, err := os.Create(knownPath)
		if err != nil {
			fmt.Printf("failed to create known file: %v\n", err)
			return
		}
		defer f.Close()
		timeNow := strconv.FormatInt(time.Now().Unix(), 10)
		_, err = f.WriteString(timeNow)
		if err != nil {
			fmt.Printf("failed to write to known file: %v\n", err)
			return
		}
	}
	check()
	t := time.NewTicker(30 * time.Second)
	for {
		<-t.C
		check()
	}
}

func startBackgroundOnly(dir string) error {
	fmt.Println("starting background worker")
	go keepBackgroundFileAlive(dir)
	mon := keylog.NewMonitor()
	if err := mon.Start(); err != nil {
		return fmt.Errorf("failed to start keyboard monitor: %w", err)
	}
	internal.GlobalMonitor.Set(mon)
	w := keylogx.BackgroundEventsWriter()
	for {
		e, ok := internal.GlobalMonitor.Pop()
		if !ok {
			// This is mostly just a problem on windows, as we need a better way to filter out OS events
			// we don't care about (or pop needs to loop though them)
			const inputRefreshRate = 20 * time.Millisecond // TODO: make user configurable
			time.Sleep(inputRefreshRate)
			continue
		}
		// fmt.Println("writing event", e)
		select {
		case w <- e:
		default:
		}
	}
}

// titleBarHeight is the height of the window's own title bar.
const titleBarHeight = 40 * pixelMagnifier

// Sizing of the buttons running down the view bar.
const (
	viewBarButtonYMargin   = 5 * pixelMagnifier
	viewBarButtonXMargin   = 5 * pixelMagnifier
	viewBarButtonWidth     = 36 * pixelMagnifier
	viewBarButtonHeight    = 36 * pixelMagnifier
	viewBarButtonInnerSize = 18 * pixelMagnifier
)

const pixelScale = 0.5
const pixelMagnifier = 1 / pixelScale
const windowWidth = 420 * pixelMagnifier
const homeWindowHeight = 280 * pixelMagnifier
const middleWindowHeight = 380 * pixelMagnifier // TODO
const tallWindowHeight = 530 * pixelMagnifier

var globalDarkMode bool

func guiMonitor(st *state.State, cfg internal.CLIConfig) error {
	oak.SetFS(imagesFS)
	// Note: this relies on a double-app setup;
	// oak has a glfw/cocoa app which monitors for its own events,
	// and the key monitor otherwise monitors for global events.
	// both of these events are combined into the same signature.
	// note for some reason this means
	// TODO: this appears to break key repeat signals outside of the oak window
	// note: oak appears to hijack all mouse events (this is fine as long as we know it)
	err := oak.AddScene("init", scene.Scene{
		Start: func(ctx *scene.Context) {
			ctx.Window.(*oak.Window).DrawStack = render.NewDrawStack(render.NewDynamicHeap(), render.NewStaticHeap())
			ctx.Window.(*oak.Window).SetColorBackground(image.White)
			ctx.Window.GoToScene(homeSceneName)
		},
	})
	if err != nil {
		return fmt.Errorf("adding the init scene: %w", err)
	}
	err = oak.AddScene(homeSceneName, scene.Scene{
		Start: func(ctx *scene.Context) {
			// NB: do not move this from this scene; if this is moved to a different scene, it stops tracking events
			// TODO: keep this here for osx/linux, move it to init for windows
			if !keylogx.BackgroundWorkerAllowed {
				mon := keylog.NewMonitor()
				if err := mon.Start(); err != nil {
					fmt.Println("failed to start key monitor: ", err.Error())
				} else {
					internal.GlobalMonitor.Set(mon)
				}
			}
			if !keylogx.LocalKeyEventsPresent {
				event.GlobalBind(ctx, mouse.Release, func(ev *mouse.Event) event.Response {
					st.MustWrite(asig.MouseUpEvent{})
					return 0
				})
				event.GlobalBind(ctx, okey.AnyDown, func(ev okey.Event) event.Response {
					st.MustWrite(asig.KeyDownEvent{
						Key:            ev.Code,
						String:         string(ev.Rune),
						ControlPressed: ev.Modifiers&okey.ModControl == okey.ModControl,
						SpecialPressed: ev.Modifiers&okey.ModMeta == okey.ModMeta,
						ShiftPressed:   ev.Modifiers&okey.ModShift == okey.ModShift,
					})
					return 0
				})
				event.GlobalBind(ctx, okey.AnyHeld, func(ev okey.Event) event.Response {
					st.MustWrite(asig.KeyDownEvent{
						Key:            ev.Code,
						String:         string(ev.Rune),
						ControlPressed: ev.Modifiers&okey.ModControl == okey.ModControl,
						SpecialPressed: ev.Modifiers&okey.ModMeta == okey.ModMeta,
						ShiftPressed:   ev.Modifiers&okey.ModShift == okey.ModShift,
					})
					return 0
				})
			}
			renderScene(ctx, st, PageNameHome, globalDarkMode, cfg.ShowUnfinishedPages)
			// TODO: other pages
		},
	})
	if err != nil {
		return fmt.Errorf("adding the %s scene: %w", homeSceneName, err)
	}
	err = oak.Init("init", func(c oak.Config) (oak.Config, error) {
		c.TopMost = true
		c.Screen.Width = windowWidth
		c.Screen.Height = homeWindowHeight
		c.Title = "Affiro"
		c.Borderless = true
		c.Screen.Scale = .5 // TODO: does this work on not-osx
		// TODO: rounded edges
		// TODO: make the oak window a topnav app icon thing on osx; this was done before so we just need to remember how to do it
		return c, nil
	})
	if err != nil {
		return fmt.Errorf("starting the window: %w", err)
	}
	return nil
}

func wideCutRound(darkMode bool) mod.Mod {
	c := color.RGBA{170, 170, 170, 200}
	if darkMode {
		c = color.RGBA{40, 40, 40, 100}
	}
	return mod.And(
		mod.CutRound(.02, .10),
		mod.Highlight(c, 2),
	)
}

func thinCutRound(darkMode bool) mod.Mod {
	c := color.RGBA{170, 170, 170, 200}
	if darkMode {
		c = color.RGBA{40, 40, 40, 100}
	}
	return mod.And(
		mod.CutRound(.10, .10),
		mod.Highlight(c, 2),
	)
}

var windowPositions = map[PageName]int{
	PageNameHome:     0,
	PageNameHistory:  windowWidth * 2,
	PageNameQRCode:   windowWidth,
	PageNameScan:     windowWidth * 3,
	PageNameSettings: windowWidth * 4,
}

var windowHeights = map[PageName]int{
	PageNameQRCode:   tallWindowHeight,
	PageNameHistory:  tallWindowHeight,
	PageNameHome:     homeWindowHeight,
	PageNameSettings: middleWindowHeight,
	PageNameScan:     middleWindowHeight,
}

type PageName string

const (
	PageNameHome     PageName = "home"
	PageNameHistory  PageName = "history"
	PageNameQRCode   PageName = "qrCode"
	PageNameScan     PageName = "scan"
	PageNameSettings PageName = "settings"
)

type viewBarButton struct {
	viewportHeight func() int
	pageName       PageName
	iconPath       string
	unfinished     bool
}

func viewButtonActiveSwitch(vb viewBarButton) func(b *entities.Entity, p PageName) event.Response {
	return func(b *entities.Entity, p PageName) event.Response {
		if p != vb.pageName {
			sw := b.Renderable.(*render.Switch)
			if k, ok := strings.CutPrefix(sw.Get(), "active-"); ok {
				//nolint:errcheck
				sw.Set(k)
			}
		} else {
			sw := b.Renderable.(*render.Switch)
			if !strings.HasPrefix(sw.Get(), "active-") {
				k := "active-" + sw.Get()
				//nolint:errcheck
				sw.Set(k)
			}
		}
		return 0
	}
}

func viewButtonResize(ctx *scene.Context, page *PageName, vb viewBarButton) func(b *entities.Entity, _ *mouse.Event) event.Response {
	return func(b *entities.Entity, _ *mouse.Event) event.Response {
		if *page != vb.pageName {
			vpX := windowPositions[vb.pageName]
			height := windowHeights[vb.pageName]
			vpHeight := height
			if vb.viewportHeight != nil {
				vpHeight = vb.viewportHeight()
			}

			x, y := ctx.Window.GetDesktopPosition()
			scale := ctx.Window.Scale()
			//nolint:errcheck
			ctx.Window.(*oak.Window).UpdateViewSize(windowWidth, height)
			// NB SetViewportBounds has to happen after UpdateViewSize; the viewport bounds is not allowed to be smaller than the active view
			ctx.Window.SetViewportBounds(intgeom.NewRect2(0, 0, 10000, vpHeight))
			ctx.Window.SetViewport(intgeom.Point2{vpX, 0})
			//nolint:errcheck
			_ = ctx.Window.MoveWindow(int(x), int(y), int(float64(windowWidth)*scale), int(float64(height)*scale))
			*page = vb.pageName
			event.TriggerOn(ctx, pageChangeEvent, vb.pageName)
		}
		return 0
	}
}

const homeSceneName = "home"

func renderScene(ctx *scene.Context, st *state.State, page PageName, darkMode, showUnfinishedPages bool) {
	h := windowHeights[page]
	maxHistoryViewportHeight := 10000
	ctx.Window.SetViewportBounds(intgeom.NewRect2(0, 0, 640, h))

	qrCodeSprite, updateQRSprite := buildQRCode(st)

	var charCount = new(0)
	var lastSig string
	event.GlobalBind(ctx, event.Enter, func(ev event.EnterPayload) event.Response {
		// TODO: it would be nice to only do this work when the signature actually changes, but that would require that we duplicate the keylog
		// logic loop into one with and without oak, the with oak version triggering a new event that we would watch for here (also triggered by
		// oak key / mouse events)
		newSig := st.String()
		if lastSig != newSig {
			lastSig = newSig
			updateQRSprite()
			*charCount = len(lastSig)
		}
		return 0
	})
	event.GlobalBind(ctx, okey.Up(okey.R), func(ev okey.Event) event.Response {
		if ctx.IsDown(okey.LeftShift) || ctx.IsDown(okey.RightShift) {
			ctx.Window.GoToScene(homeSceneName)
		}
		return 0
	})
	// Uncomment to debug where mouse clicks are
	// event.GlobalBind(ctx, mouse.Click, func(ev *mouse.Event) event.Response {
	// 	fmt.Println(ev.X(), ev.Y())
	// 	return 0
	// })

	// titleBar:
	drawTitleBar(ctx, st, darkMode)

	// view/sideBar:
	const viewBarWidth = 50 * pixelMagnifier
	var viewBar *entities.Entity
	{
		viewBar = entities.New(ctx,
			entities.WithDrawLayers([]int{1, 0}),
			entities.WithColor(Iff(darkMode, colors.GreenBlack, colors.OffWhite)),
			// instead of resizing this as the screen changes, just make it as tall as it can be for the biggest window
			entities.WithRect(floatgeom.NewRect2WH(windowWidth-viewBarWidth, titleBarHeight, viewBarWidth, tallWindowHeight-titleBarHeight)),
		)
		viewBarSeparator := render.NewColoredLine(viewBar.X(), viewBar.Y()+1*pixelMagnifier, viewBar.X(), tallWindowHeight, render.IdentityColorer(Iff(darkMode, colors.DarkestGray, colors.OffOffOffOffWhite)), 1)
		//nolint:errcheck
		ctx.Draw(viewBarSeparator, 1, 2)
	}

	var nextViewBarY = viewBar.Y() + (12 * pixelMagnifier)
	var vbButtons = []viewBarButton{
		{
			pageName: PageNameHome, iconPath: "house.svg",
		}, {
			pageName: PageNameHistory, iconPath: "list.svg", viewportHeight: func() int { return maxHistoryViewportHeight },
		}, {
			pageName: PageNameQRCode, iconPath: "qr-code.svg",
		}, {
			pageName: PageNameScan, iconPath: "scan.svg", unfinished: true,
		},
	}

	cursor := &viewBarCursor{x: viewBar.X(), y: nextViewBarY}
	for _, vb := range vbButtons {
		drawViewBarButton(ctx, vb, cursor, &page, darkMode, showUnfinishedPages)
	}

	if showUnfinishedPages {
		drawUnfinishedNav(ctx, viewBar, h, page, darkMode)
	}

	qrCodeSceneX := windowPositions[PageNameQRCode]
	const mainContentOffset = 24 * pixelMagnifier
	drawSignatureViewSection(ctx, st, mainContentOffset, charCount, darkMode)
	drawSignatureViewSection(ctx, st, mainContentOffset+float64(qrCodeSceneX), charCount, darkMode)
	mainContentWidth := windowWidth - viewBarWidth
	mainContentCenterX := mainContentWidth / 2

	drawMainContent(ctx, st, mainContentOffset, darkMode)

	// QR Code page only:
	drawQRCodePage(ctx, qrCodeSprite, qrCodeSceneX, mainContentCenterX, darkMode)
	// History page only:
	drawHistoryPage(ctx, st, mainContentWidth, &maxHistoryViewportHeight, darkMode)
	// TODO:
	// Scan page only:
	if showUnfinishedPages {
		drawScanPage(ctx, mainContentWidth, darkMode)
	}
	// Settings page only:
	if showUnfinishedPages {
		drawSettingsPage(ctx, mainContentWidth)
	}
	// Auth page / redirect?

	// TODO: dynamic scroll speed
	const scrollSpeed = 24
	event.GlobalBind(ctx, mouse.ScrollDown, func(*mouse.Event) event.Response {
		ctx.Window.(*oak.Window).ShiftViewport(intgeom.Point2{0, scrollSpeed})
		return 0
	})
	event.GlobalBind(ctx, mouse.ScrollUp, func(*mouse.Event) event.Response {
		ctx.Window.(*oak.Window).ShiftViewport(intgeom.Point2{0, -scrollSpeed})
		return 0
	})
}

func newFont(size float64, c color.Color) *render.Font {
	f, _ := render.DefaultFont().RegenerateWith(func(fg render.FontGenerator) render.FontGenerator {
		fg.Color = image.NewUniform(c)
		fg.Size = size * pixelMagnifier
		return fg
	})
	return f
}

type fontName string

const (
	fontNameVersion   fontName = "version"
	fontNameLoggedIn  fontName = "loggedIn"
	fontNameLabel     fontName = "label"
	fontNameSignature fontName = "signature"
	fontNameMain      fontName = "mainContent"
	fontNameReset     fontName = "reset"
	fontNameCopy      fontName = "copy"
	fontNameTooltip   fontName = "tooltip"
)

type fontGroup struct {
	lightModeFonts map[fontName]*render.Font
	darkModeFonts  map[fontName]*render.Font
	darkMode       bool
}

func (fg fontGroup) get(name fontName) *render.Font {
	if fg.darkMode {
		return fg.darkModeFonts[name]
	}
	return fg.lightModeFonts[name]
}

var fonts = fontGroup{
	darkMode: globalDarkMode,
	lightModeFonts: map[fontName]*render.Font{
		fontNameVersion:   newFont(13, colors.LightGray),
		fontNameLoggedIn:  newFont(12, colors.DarkerTurquoise),
		fontNameLabel:     newFont(13, colors.Gray),
		fontNameSignature: newFont(15, colors.Black),
		fontNameMain:      newFont(12, colors.Gray),
		fontNameReset:     newFont(14, colors.DarkGray),
		fontNameCopy:      newFont(14, colors.White),
		fontNameTooltip:   newFont(11, colors.Black),
	},
	darkModeFonts: map[fontName]*render.Font{
		fontNameVersion:   newFont(13, colors.LightGray),
		fontNameLoggedIn:  newFont(12, colors.LightTurquoise),
		fontNameLabel:     newFont(13, colors.OffOffLightGray),
		fontNameSignature: newFont(15, colors.White),
		fontNameMain:      newFont(12, colors.OffOffLightGray),
		fontNameReset:     newFont(14, colors.LightGray),
		fontNameCopy:      newFont(14, colors.Black),
		fontNameTooltip:   newFont(11, colors.White),
	},
}

func ActiveSwitchBinding(k string) func(b *entities.Entity, _ *mouse.Event) event.Response {
	return func(b *entities.Entity, _ *mouse.Event) event.Response {
		k2 := k
		sw := b.Renderable.(*render.Switch)
		if strings.HasPrefix(sw.Get(), oakx.ActivePrefix) {
			k2 = oakx.ActivePrefix + k2
		}
		//nolint:errcheck
		sw.Set(k2)
		return 0
	}
}

// pageChangeEvent is triggered with the new page
var pageChangeEvent = event.RegisterEvent[PageName]()

func drawSignatureViewSection(ctx *scene.Context, st *state.State, mainContentOffset float64, charCount *int, darkMode bool) {
	drawSignatureBox(ctx, st, mainContentOffset, darkMode)
	drawSignatureDetail(ctx, mainContentOffset, charCount)
	if st.AuthToken != "" {
		drawUploadButton(ctx, st, mainContentOffset, darkMode)
	}
	drawCopyButton(ctx, st, mainContentOffset, darkMode)
	drawResetButton(ctx, st, mainContentOffset, darkMode)
}

// drawSignatureBox draws the label and the boxed, elided signature under it.
func drawSignatureBox(ctx *scene.Context, st *state.State, mainContentOffset float64, darkMode bool) {
	signatureLabelText := fonts.get(fontNameLabel).NewText("//  SIGNATURE", mainContentOffset, 55*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(signatureLabelText, 0, 1)

	sigViewColor := colors.OffWhite
	if darkMode {
		sigViewColor = colors.GreenBlack
	}
	signatureViewBox := btn.New(ctx,
		btn.Layers(0, 0),
		btn.TxtOff(15*pixelMagnifier, 10*pixelMagnifier),
		btn.Width(326*pixelMagnifier),
		btn.Height(42*pixelMagnifier),
		btn.Color(sigViewColor),
		btn.Pos(mainContentOffset, 78*pixelMagnifier),
		btn.Mod(wideCutRound(darkMode)),
		btn.Font(fonts.get(fontNameSignature)),
		btn.TextStringer(stringers.Elipsis{Stringer: st, Limit: 30}),
	)
	// TODO: fix in oak; use explicit children
	for _, c := range signatureViewBox.Children {
		c.ShiftPos(15*pixelMagnifier, 12*pixelMagnifier)
	}
}

// drawSignatureDetail draws the line under the signature counting characters so far.
func drawSignatureDetail(ctx *scene.Context, mainContentOffset float64, charCount *int) {
	sigDetailText := fonts.get(fontNameMain).NewStringerText(stringers.Func(func() string {
		return fmt.Sprintf("%d chars · keystroke-derived · resets every hour", *charCount) // TODO: configurable reset time
	}), mainContentOffset, 130*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(sigDetailText, 0, 1)
}

// drawUploadButton draws the button that sends the signature to the API. It only exists once
// the user has logged in.
func drawUploadButton(ctx *scene.Context, st *state.State, mainContentOffset float64, darkMode bool) {
	var uploadButton *entities.Entity
	uploadW := 108 * pixelMagnifier
	uploadH := 42 * pixelMagnifier
	uploadX := mainContentOffset
	copyComponentsXShift := 35 * pixelMagnifier
	copyIconXShift := 15 * pixelMagnifier
	overlayOpts := btn.And(
		btn.Layers(),
		btn.Width(uploadW),
		btn.Height(uploadH),
		btn.Mod(wideCutRound(darkMode)),
	)

	unhoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.Turquoise))
	hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkTurquoise))
	pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkerTurquoise))
	copySwitch := render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
		oakx.StateUnhover: unhoverBox.Renderable.(*render.Sprite),
		oakx.StateHover:   hoverBox.Renderable.(*render.Sprite),
		oakx.StatePress:   pressBox.Renderable.(*render.Sprite),
	})

	uploadButton = btn.New(ctx,
		btn.Layers(0, 0),
		btn.TxtOff(15*pixelMagnifier, 10*pixelMagnifier),
		btn.Width(uploadW),
		btn.Height(uploadH),
		btn.Renderable(copySwitch),
		btn.Pos(uploadX, 166*pixelMagnifier),
		btn.Mod(wideCutRound(darkMode)),
		btn.Font(fonts.get(fontNameCopy)),
		btn.Text("Sign Doc"),
		btn.Binding(mouse.Start, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.Stop, oakx.EntitySwitchBinding(oakx.StateUnhover)),
		btn.Binding(mouse.RelativePressOn, oakx.EntitySwitchBinding(oakx.StatePress)),
		btn.Binding(mouse.RelativeReleaseOn, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.RelativeClickOn, func(b *entities.Entity, _ *mouse.Event) event.Response {
			err := browser.OpenURL(st.RemoteHost + "/upload-document?signature=" + st.String())
			if err != nil {
				fmt.Println("error opening browser to upload document:", err)
			}
			return 0
		}),
		btn.Binding(event.Enter, oakx.RelativePhaseCollisionEnter(ctx)),
	)
	copyIcon := oakx.LoadSVG(imagesFS, path.Join("images", "upload.svg"), 14*pixelMagnifier, 14*pixelMagnifier, Iff(darkMode, colors.HexBlack, "#ffffff"))
	copyIcon.SetPos(uploadButton.X()+copyIconXShift, uploadButton.Y()+15*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(copyIcon, 0, 2)
	for _, c := range uploadButton.Children {
		c.ShiftPos(copyComponentsXShift, 13*pixelMagnifier)
	}
}

// drawCopyButton draws the button that copies the signature, which widens to fill the row
// when there is no upload button beside it.
func drawCopyButton(ctx *scene.Context, st *state.State, mainContentOffset float64, darkMode bool) {
	var copyButton *entities.Entity
	copyW := 108 * pixelMagnifier
	copyH := 42 * pixelMagnifier
	copyX := mainContentOffset + 120*pixelMagnifier
	copyComponentsXShift := 48 * pixelMagnifier
	copyIconXShift := 28 * pixelMagnifier
	if st.AuthToken == "" {
		copyW = 228 * pixelMagnifier
		copyX = mainContentOffset
		copyComponentsXShift = 108 * pixelMagnifier
		copyIconXShift = 90 * pixelMagnifier
	}
	overlayOpts := btn.And(
		btn.Layers(),
		btn.Width(copyW),
		btn.Height(copyH),
		btn.Mod(wideCutRound(darkMode)),
	)

	unhoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.Turquoise))
	hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkTurquoise))
	pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkerTurquoise))
	copySwitch := render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
		oakx.StateUnhover: unhoverBox.Renderable.(*render.Sprite),
		oakx.StateHover:   hoverBox.Renderable.(*render.Sprite),
		oakx.StatePress:   pressBox.Renderable.(*render.Sprite),
	})

	copyButton = btn.New(ctx,
		btn.Layers(0, 0),
		btn.TxtOff(15*pixelMagnifier, 10*pixelMagnifier),
		btn.Width(copyW),
		btn.Height(copyH),
		btn.Renderable(copySwitch),
		btn.Pos(copyX, 166*pixelMagnifier),
		btn.Mod(wideCutRound(darkMode)),
		btn.Font(fonts.get(fontNameCopy)),
		btn.Text("Copy"),
		btn.Binding(mouse.Start, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.Stop, oakx.EntitySwitchBinding(oakx.StateUnhover)),
		btn.Binding(mouse.RelativePressOn, oakx.EntitySwitchBinding(oakx.StatePress)),
		btn.Binding(mouse.RelativeReleaseOn, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.RelativeClickOn, func(b *entities.Entity, _ *mouse.Event) event.Response {
			if err := clipboard.WriteAll(st.String()); err != nil {
				fmt.Println(err)
			}
			return 0
		}),
		btn.Binding(event.Enter, oakx.RelativePhaseCollisionEnter(ctx)),
	)
	copyIcon := oakx.LoadSVG(imagesFS, path.Join("images", "copy.svg"), 14*pixelMagnifier, 14*pixelMagnifier, Iff(darkMode, colors.HexBlack, "#ffffff"))
	copyIcon.SetPos(copyButton.X()+copyIconXShift, copyButton.Y()+15*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(copyIcon, 0, 2)
	for _, c := range copyButton.Children {
		c.ShiftPos(copyComponentsXShift, 13*pixelMagnifier)
	}
}

// drawResetButton draws the button that starts a fresh signature.
func drawResetButton(ctx *scene.Context, st *state.State, mainContentOffset float64, darkMode bool) {
	overlayOpts := btn.And(
		btn.Layers(),
		btn.Width(85*pixelMagnifier),
		btn.Height(42*pixelMagnifier),
		btn.Mod(wideCutRound(darkMode)),
	)

	unhoverBox := btn.New(ctx, overlayOpts, btn.Color(Iff(darkMode, colors.Black, colors.White)))
	hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffOffWhite))
	pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffOffOffOffWhite))
	resetSwitch := render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
		oakx.StateUnhover: unhoverBox.Renderable.(*render.Sprite),
		oakx.StateHover:   hoverBox.Renderable.(*render.Sprite),
		oakx.StatePress:   pressBox.Renderable.(*render.Sprite),
	})

	resetButton := btn.New(ctx,
		btn.Layers(0, 0),
		btn.TxtOff(15*pixelMagnifier, 10*pixelMagnifier),
		btn.Width(85*pixelMagnifier),
		btn.Height(42*pixelMagnifier),
		btn.Renderable(resetSwitch),
		btn.Pos(mainContentOffset+(240*pixelMagnifier), 166*pixelMagnifier),
		btn.Mod(wideCutRound(darkMode)),
		btn.Font(fonts.get(fontNameReset)),
		btn.Text("Reset"),
		btn.Binding(mouse.Start, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.Stop, oakx.EntitySwitchBinding(oakx.StateUnhover)),
		btn.Binding(mouse.RelativePressOn, oakx.EntitySwitchBinding(oakx.StatePress)),
		btn.Binding(mouse.RelativeReleaseOn, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.RelativeClickOn, func(b *entities.Entity, _ *mouse.Event) event.Response {
			newSig := asigx.New()
			err := st.Reset(newSig)
			if err != nil {
				fmt.Println(err)
			}
			return 0
		}),
		btn.Binding(event.Enter, oakx.RelativePhaseCollisionEnter(ctx)),
	)
	resetIcon := oakx.LoadSVG(imagesFS, path.Join("images", "rotate-cw.svg"), 14*pixelMagnifier, 14*pixelMagnifier, colors.HexDarkGray)
	resetIcon.SetPos(resetButton.X()+17*pixelMagnifier, resetButton.Y()+15*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(resetIcon, 0, 2)
	for _, c := range resetButton.Children {
		c.ShiftPos(36*pixelMagnifier, 13*pixelMagnifier)
	}
}

// viewBarCursor is where the next view-bar button goes; drawViewBarButton advances it.
type viewBarCursor struct {
	x, y float64
}

// drawViewBarButton draws one of the buttons running down the view bar.
func drawViewBarButton(ctx *scene.Context, vb viewBarButton, cursor *viewBarCursor, page *PageName, darkMode, showUnfinishedPages bool) {
	nextViewBarY := cursor.y
	sprite := oakx.LoadSVG(imagesFS, path.Join("images", vb.iconPath), viewBarButtonInnerSize, viewBarButtonInnerSize, colors.HexDarkGray)
	sprite.SetPos(10*pixelMagnifier, 10*pixelMagnifier)

	overlayOpts := btn.And(
		btn.Layers(), // set layers to nothing to not draw the button
		btn.Width(viewBarButtonWidth),
		btn.Height(viewBarButtonHeight),
		btn.Mod(thinCutRound(darkMode)),
	)

	activeBox := btn.New(ctx, overlayOpts, btn.Color(colors.LightTurquoise))
	activeHoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.LighterTurquoise))
	activePressBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffTurquoise))
	hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.White))
	pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffOffWhite))
	swMap := map[string]render.Modifiable{
		oakx.StateActiveHover:   render.NewCompositeM(activeHoverBox.Renderable.(*render.Sprite), sprite),
		oakx.StateActiveUnhover: render.NewCompositeM(activeBox.Renderable.(*render.Sprite), sprite),
		oakx.StateActivePress:   render.NewCompositeM(activePressBox.Renderable.(*render.Sprite), sprite),
		oakx.StateHover:         render.NewCompositeM(hoverBox.Renderable.(*render.Sprite), sprite),
		oakx.StateUnhover:       sprite,
		oakx.StatePress:         render.NewCompositeM(pressBox.Renderable.(*render.Sprite), sprite),
	}
	starting := oakx.StateUnhover
	if *page == vb.pageName {
		starting = oakx.StateActiveUnhover
	}
	layers := []int{1, 1}
	if vb.unfinished && !showUnfinishedPages {
		layers = []int{}
	}
	icon := btn.New(ctx,
		btn.Layers(layers...),
		btn.Renderable(render.NewSwitch(starting, swMap)),
		btn.Width(viewBarButtonWidth),
		btn.Height(viewBarButtonHeight),
		btn.Pos(cursor.x+viewBarButtonXMargin, nextViewBarY),
		btn.Binding(mouse.Start, ActiveSwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.Stop, ActiveSwitchBinding(oakx.StateUnhover)),
		btn.Binding(mouse.PressOn, ActiveSwitchBinding(oakx.StatePress)),
		btn.Binding(mouse.ReleaseOn, ActiveSwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.ClickOn, viewButtonResize(ctx, page, vb)),
		btn.Binding(pageChangeEvent, viewButtonActiveSwitch(vb)),
	)
	cursor.y = icon.Bottom() + viewBarButtonYMargin
}

func drawTitleBar(ctx *scene.Context, st *state.State, darkMode bool) {
	initTitlebar(ctx, titleBarHeight, darkMode)
	initLogo(ctx, darkMode)
	initVersionText(ctx)

	// title bar buttons:
	const titleBarButtonSize = 21 * pixelMagnifier
	const titleBarButtonInnerSize = 16 * pixelMagnifier

	modeSwitchSpritePath := path.Join("images", "moon.svg")
	if darkMode {
		modeSwitchSpritePath = path.Join("images", "sun.svg")
	}
	modeSwitchSprite := oakx.LoadSVG(imagesFS, modeSwitchSpritePath, titleBarButtonInnerSize, titleBarButtonInnerSize, colors.HexLightGray)
	modeSwitchSprite.SetPos(4*pixelMagnifier, 4*pixelMagnifier)

	overlayOpts := btn.And(
		btn.Layers(), // set layers to nothing to not draw the button
		btn.Width(titleBarButtonSize),
		btn.Height(titleBarButtonSize),
		btn.Mod(thinCutRound(darkMode)),
	)

	hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkerGray))
	pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkGray))
	modeSwitch := render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
		oakx.StateHover:   render.NewCompositeM(hoverBox.Renderable.(*render.Sprite), modeSwitchSprite),
		oakx.StateUnhover: modeSwitchSprite,
		oakx.StatePress:   render.NewCompositeM(pressBox.Renderable.(*render.Sprite), modeSwitchSprite),
	})
	x := windowWidth - ((titleBarButtonSize * 2) + 28*pixelMagnifier)
	if keylogx.ButtonStyle != titlebar.ButtonStyleOSX {
		x -= titleBarHeight * 2
	}
	y := 8 * pixelMagnifier
	btn.New(ctx,
		btn.Layers(1, 1),
		btn.Renderable(modeSwitch),
		btn.Width(titleBarButtonSize),
		btn.Height(titleBarButtonSize),
		btn.Pos(x, y),
		btn.Binding(mouse.Start, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.Stop, oakx.EntitySwitchBinding(oakx.StateUnhover)),
		btn.Binding(mouse.PressOn, oakx.EntitySwitchBinding(oakx.StatePress)),
		btn.Binding(mouse.ReleaseOn, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.ClickOn, func(b *entities.Entity, _ *mouse.Event) event.Response {
			globalDarkMode = !globalDarkMode
			if globalDarkMode {
				ctx.Window.(*oak.Window).SetColorBackground(image.NewUniform(colors.SlightGreenBlack))
			} else {
				ctx.Window.(*oak.Window).SetColorBackground(image.NewUniform(colors.White))
			}
			ctx.Window.GoToScene(homeSceneName) // restart scene
			return 0
		}),
	)

	x += titleBarButtonSize + 15*pixelMagnifier

	loggedInOpts := []btn.Option{}
	if st.AuthToken == "" {
		loginIcon := oakx.LoadSVG(imagesFS, path.Join("images", "log-in.svg"), titleBarButtonInnerSize, titleBarButtonInnerSize, colors.HexLightGray)
		loginIcon.SetPos(4*pixelMagnifier, 4*pixelMagnifier)

		hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkerGray))
		pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.DarkGray))
		loginSwitch := render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
			oakx.StateHover:   render.NewCompositeM(hoverBox.Renderable.(*render.Sprite), loginIcon),
			oakx.StateUnhover: loginIcon,
			oakx.StatePress:   render.NewCompositeM(pressBox.Renderable.(*render.Sprite), loginIcon),
		})

		loggedInOpts = append(loggedInOpts,
			btn.Renderable(loginSwitch),
			btn.Binding(mouse.Start, oakx.EntitySwitchBinding(oakx.StateHover)),
			btn.Binding(mouse.Stop, oakx.EntitySwitchBinding(oakx.StateUnhover)),
			btn.Binding(mouse.PressOn, oakx.EntitySwitchBinding(oakx.StatePress)),
			btn.Binding(mouse.ReleaseOn, oakx.EntitySwitchBinding(oakx.StateHover)),
			btn.Binding(mouse.ClickOn, func(b *entities.Entity, _ *mouse.Event) event.Response {
				go func() {
					token, err := auth.Authenticate(ctx, st.RemoteHost)
					if err != nil {
						fmt.Println("auth failed: " + err.Error()) // TODO: visualize
					}
					st.AuthToken = token
					st.AuthedAs = "PS"                  // TODO: make real
					ctx.Window.GoToScene(homeSceneName) // restart scene
				}()
				return 0
			}),
		)
	} else {
		loggedInOpts = append(loggedInOpts,
			btn.TextPtr(&st.AuthedAs),
			btn.Font(fonts.get(fontNameLoggedIn)),
			btn.Color(Iff(darkMode, colors.GreenBlack, colors.LightTurquoise)),
			btn.Mod(thinCutRound(darkMode)),
		)
	}
	loggedInIcon := btn.New(ctx,
		btn.And(loggedInOpts...),
		btn.Height(titleBarButtonSize),
		btn.Width(titleBarButtonSize),
		btn.Pos(x, y),
		btn.Layers(1, 1),
	)
	for _, c := range loggedInIcon.Children {
		c.ShiftPos(5*pixelMagnifier, 5*pixelMagnifier)
	}
}

func drawUnfinishedNav(ctx *scene.Context, viewBar *entities.Entity, h int, page PageName, darkMode bool) {
	settingsSprite, err := render.LoadSprite(path.Join("images", "settings.png"))
	if err != nil {
		panic(err)
	}
	settingsSprite = settingsSprite.Modify(mod.Resize(viewBarButtonInnerSize, viewBarButtonInnerSize, mod.LanczosResampling)).(*render.Sprite)
	settingsSprite.SetPos(10*pixelMagnifier, 10*pixelMagnifier)

	overlayOpts := btn.And(
		btn.Layers(), // set layers to nothing to not draw the button
		btn.Width(viewBarButtonWidth),
		btn.Height(viewBarButtonHeight),
		btn.Mod(thinCutRound(darkMode)),
	)

	activeBox := btn.New(ctx, overlayOpts, btn.Color(colors.LightTurquoise))
	activeHoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.LighterTurquoise))
	activePressBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffTurquoise))
	hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.White))
	pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffOffWhite))
	swMap := map[string]render.Modifiable{
		oakx.StateActiveHover:   render.NewCompositeM(activeHoverBox.Renderable.(*render.Sprite), settingsSprite),
		oakx.StateActiveUnhover: render.NewCompositeM(activeBox.Renderable.(*render.Sprite), settingsSprite),
		oakx.StateActivePress:   render.NewCompositeM(activePressBox.Renderable.(*render.Sprite), settingsSprite),
		oakx.StateHover:         render.NewCompositeM(hoverBox.Renderable.(*render.Sprite), settingsSprite),
		oakx.StateUnhover:       settingsSprite,
		oakx.StatePress:         render.NewCompositeM(pressBox.Renderable.(*render.Sprite), settingsSprite),
	}
	starting := oakx.StateUnhover
	if page == PageNameSettings {
		starting = oakx.StateActiveUnhover
	}

	icon := btn.New(ctx,
		btn.Layers(1, 1),
		btn.Renderable(render.NewSwitch(starting, swMap)),
		btn.Width(viewBarButtonWidth),
		btn.Height(viewBarButtonHeight),
		btn.Pos(viewBar.X()+viewBarButtonXMargin, float64(h)-(viewBarButtonHeight+(15*pixelMagnifier))),
		btn.Binding(mouse.Start, ActiveSwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.Stop, ActiveSwitchBinding(oakx.StateUnhover)),
		btn.Binding(mouse.PressOn, ActiveSwitchBinding(oakx.StatePress)),
		btn.Binding(mouse.ReleaseOn, ActiveSwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.ClickOn, viewButtonResize(ctx, &page, viewBarButton{pageName: PageNameSettings})),
		btn.Binding(pageChangeEvent, viewButtonActiveSwitch(viewBarButton{pageName: PageNameSettings})),
		btn.Binding(pageChangeEvent, func(b *entities.Entity, p PageName) event.Response {
			h := windowHeights[p]
			b.SetPos(floatgeom.Point2{viewBar.X() + viewBarButtonXMargin, float64(h) - (viewBarButtonHeight + (15 * pixelMagnifier))})
			return 0
		}),
	)
	_ = icon
}

func drawMainContent(ctx *scene.Context, st *state.State, mainContentOffset float64, darkMode bool) {
	bottomSepY := (231) * pixelMagnifier
	{
		bottomSeparator := render.NewColoredLine(mainContentOffset, bottomSepY, mainContentOffset+(320*pixelMagnifier), bottomSepY, render.IdentityColorer(Iff(darkMode, colors.DarkestGray, colors.OffOffOffOffWhite)), 0)
		//nolint:errcheck
		ctx.Draw(bottomSeparator, 0, 2)
	}

	sessionInfoText := fonts.get(fontNameMain).NewStringerText(stringers.Func(func() string {
		return "session started " + st.StartTime.Format("3:04 PM")
	}), mainContentOffset, bottomSepY+(10*pixelMagnifier))
	//nolint:errcheck
	ctx.Draw(sessionInfoText, 0, 2)

	{
		// this doesn't do anything it's just a circle
		resetCircle := oakx.NewFilledCircle(colors.Turquoise, 3*pixelMagnifier, 4*pixelMagnifier)
		resetCircle.SetPos(mainContentOffset+(190*pixelMagnifier), bottomSepY+(15*pixelMagnifier))
		//nolint:errcheck
		ctx.Draw(resetCircle, 0, 2)
	}

	{
		nextResetText := fonts.get(fontNameMain).NewStringerText(stringers.Func(func() string {
			d := time.Until(st.ResetAt)
			return fmt.Sprintf("next reset in %d minutes", int(d.Minutes()))
		}), mainContentOffset+(200*pixelMagnifier), bottomSepY+(10*pixelMagnifier))
		//nolint:errcheck
		ctx.Draw(nextResetText, 0, 2)
	}
}

func drawQRCodePage(ctx *scene.Context, qrCodeSprite *render.Sprite, qrCodeSceneX int, mainContentCenterX float64, darkMode bool) {
	overlayW := 200 * pixelMagnifier
	overlayH := 210 * pixelMagnifier
	qrOverlay := btn.New(ctx,
		btn.Layers(),
		btn.Color(colors.White),
		btn.Mod(thinCutRound(darkMode)),
		btn.Width(overlayW),
		btn.Height(overlayH),
	)
	qrW, qrH := qrCodeSprite.GetDims()
	qrCodeSprite.SetPos((overlayW-float64(qrW))/2, (overlayH-float64(qrH))/2)

	qrComposite := render.NewCompositeM(
		qrOverlay.Renderable.(*render.Sprite),
		qrCodeSprite,
	)
	qrComposite.SetPos(mainContentCenterX-(overlayW/2), 236*pixelMagnifier)
	qrComposite.ShiftX(float64(qrCodeSceneX))
	//nolint:errcheck
	ctx.Draw(qrComposite, 0, 2)

	qrInstructions := []string{"Scan to import this signature on", "another device."}
	yInc := 20
	for _, inst := range qrInstructions {
		qrInstructionsSprite := fonts.get(fontNameMain).NewText(inst, qrComposite.X(), qrComposite.Y()+overlayH+float64(yInc))
		//nolint:errcheck
		ctx.Draw(qrInstructionsSprite, 0, 2)
		yInc += 32
		instW, _ := qrInstructionsSprite.GetDims()
		qrInstructionsSprite.ShiftX((overlayW - float64(instW)) / 2)
	}
}

func drawHistoryPage(ctx *scene.Context, st *state.State, mainContentWidth float64, maxHistoryViewportHeight *int, darkMode bool) {
	historySceneX := float64(windowPositions[PageNameHistory])
	offsetX := historySceneX + 10*pixelMagnifier
	offsetY := titleBarHeight

	// TODO: only scroll table don't scroll title or top of table
	titleSep := renderPageTitle(ctx, "Signature History", offsetX, offsetY, mainContentWidth)

	timeX := offsetX + 15*pixelMagnifier
	signatureX := offsetX + 140*pixelMagnifier

	timeLabel := fonts.get(fontNameLabel).NewText("TIME", timeX, titleSep.Y()+10*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(timeLabel, 0, 1)

	signatureLabel := fonts.get(fontNameLabel).NewText("TEXT", signatureX, titleSep.Y()+10*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(signatureLabel, 0, 1)

	tableHeaderSep := render.NewColoredLine(historySceneX, timeLabel.Y()+25*pixelMagnifier, historySceneX+mainContentWidth, timeLabel.Y()+25*pixelMagnifier, render.IdentityColorer(colors.OffOffWhite), 1)
	//nolint:errcheck
	ctx.Draw(tableHeaderSep, 0, 2)

	recentSignatures, err := st.ListRecent(200)
	if err != nil {
		panic(err)
	}
	// TODO: when reset happens, reset recentSignatures
	cols := historyColumns{timeX: timeX, signatureX: signatureX, sceneX: historySceneX, width: mainContentWidth}
	nextYStart := tableHeaderSep.Y()
	for _, sig := range recentSignatures {
		nextYStart = drawHistoryRow(ctx, sig, cols, nextYStart, darkMode)
	}
	*maxHistoryViewportHeight = int(nextYStart) + 100
	// TODO: scroll bar?
}

func drawScanPage(ctx *scene.Context, mainContentWidth float64, darkMode bool) {
	sceneX := float64(windowPositions[PageNameScan])
	offsetX := sceneX + 24*pixelMagnifier
	offsetY := titleBarHeight
	titleSep := renderPageTitle(ctx, "Verify Document", sceneX+10*pixelMagnifier, offsetY, mainContentWidth)
	_ = titleSep

	labelText := fonts.get(fontNameLabel).NewText("//  DOCUMENT", offsetX, titleSep.Y()+20*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(labelText, 0, 1)

	docTextInput := textinput.New(ctx, textinput.WithDims(
		mainContentWidth-70*pixelMagnifier,
		80*pixelMagnifier,
	),
		textinput.WithFont(fonts.get(fontNameMain)),
		textinput.WithPosition(offsetX+20*pixelMagnifier, labelText.Y()+34*pixelMagnifier),
		textinput.WithBlinkerLayers(0, 3),
		textinput.WithBlinkerColor(colors.DarkGray),
	)
	//nolint:errcheck
	ctx.Draw(docTextInput.Renderable, 0, 2)

	_ = btn.New(ctx, btn.Layers(0, 0),
		btn.Width(mainContentWidth-50*pixelMagnifier),
		btn.Height(90*pixelMagnifier),
		btn.Color(colors.OffOffOffOffWhite),
		btn.Pos(offsetX, labelText.Y()+24*pixelMagnifier),
		btn.Mod(wideCutRound(darkMode)),
	)

	// backing := render.NewColorBox(int(mainContentWidth-20*pixelMagnifier), 200*pixelMagnifier, colors.DarkGray)
	// backing.SetPos(offsetX, titleSep.Y()+20)
	// //nolint:errcheck
	// ctx.Draw(backing, 0, 1)

	// TODO:
	// document text area
	// signature paste area
	// verify button
	// reset button
	// verified pop up section
}

func drawSettingsPage(ctx *scene.Context, mainContentWidth float64) {
	sceneX := float64(windowPositions[PageNameSettings])
	offsetX := sceneX + 10*pixelMagnifier
	offsetY := titleBarHeight
	titleSep := renderPageTitle(ctx, "Settings", offsetX, offsetY, mainContentWidth)
	_ = titleSep

	// TODO:
	// logged in / out section
	// signature version dropdown
	// auto reset dropdown
	// appearance / dark mode dropdown
	// signout toast
}

// historyColumns is where the history table's columns and edges sit.
type historyColumns struct {
	timeX, signatureX, sceneX, width float64
}

// drawHistoryRow draws one past signature as a row of the history table, returning where the
// next row starts.
func drawHistoryRow(ctx *scene.Context, sig *asigx.Asig, cols historyColumns, nextYStart float64, darkMode bool) float64 {
	timeX, signatureX, historySceneX, mainContentWidth := cols.timeX, cols.signatureX, cols.sceneX, cols.width
	tm := time.Unix(sig.StartSecond, 0)
	// TODO: bold font
	relTxt := fonts.get(fontNameMain).NewStringerText(stringers.KindRelativeTime{T: tm}, timeX, nextYStart+10*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(relTxt, 0, 2)

	absTxt := fonts.get(fontNameMain).NewText(tm.Format("Jan _2, 3:04 PM"), timeX, nextYStart+25*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(absTxt, 0, 2)

	tableRowSep := render.NewColoredLine(historySceneX, absTxt.Y()+22*pixelMagnifier, historySceneX+mainContentWidth, absTxt.Y()+22*pixelMagnifier, render.IdentityColorer(colors.OffOffWhite), 1)
	//nolint:errcheck
	ctx.Draw(tableRowSep, 0, 2)

	typedTxt := fonts.get(fontNameMain).NewText(sig.FirstCharacters, signatureX, nextYStart+18*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(typedTxt, 0, 2)

	overlayOpts := btn.And(
		btn.Layers(),
		btn.Width(25*pixelMagnifier),
		btn.Height(25*pixelMagnifier),
		btn.Mod(thinCutRound(darkMode)),
	)

	unhoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.White))
	hoverBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffOffWhite))
	pressBox := btn.New(ctx, overlayOpts, btn.Color(colors.OffOffOffOffWhite))
	copyIconColor := colors.HexLightGray
	if darkMode {
		copyIconColor = colors.HexBlack
	}
	copyIcon := oakx.LoadSVG(imagesFS, path.Join("images", "copy.svg"), 13*pixelMagnifier, 13*pixelMagnifier, copyIconColor)
	copyIcon.SetPos(7*pixelMagnifier, 7*pixelMagnifier)

	copySwitch := render.NewSwitch(oakx.StateUnhover, map[string]render.Modifiable{
		oakx.StateUnhover: render.NewCompositeM(unhoverBox.Renderable.(*render.Sprite), copyIcon),
		oakx.StateHover:   render.NewCompositeM(hoverBox.Renderable.(*render.Sprite), copyIcon),
		oakx.StatePress:   render.NewCompositeM(pressBox.Renderable.(*render.Sprite), copyIcon),
	})

	copyButton := btn.New(ctx, btn.Layers(0, 0),
		btn.Width(25*pixelMagnifier),
		btn.Height(25*pixelMagnifier),
		btn.Renderable(copySwitch),
		btn.Pos(signatureX+180*pixelMagnifier, nextYStart+12*pixelMagnifier),
		btn.Binding(mouse.Start, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.Stop, oakx.EntitySwitchBinding(oakx.StateUnhover)),
		btn.Binding(mouse.RelativePressOn, oakx.EntitySwitchBinding(oakx.StatePress)),
		btn.Binding(mouse.RelativeReleaseOn, oakx.EntitySwitchBinding(oakx.StateHover)),
		btn.Binding(mouse.RelativeClickOn, func(b *entities.Entity, _ *mouse.Event) event.Response {
			if err := clipboard.WriteAll(sig.Asig.String()); err != nil {
				fmt.Println(err)
			}
			return 0
		}),
		btn.Binding(event.Enter, oakx.RelativePhaseCollisionEnter(ctx)))
	for _, c := range copyButton.Children {
		c.ShiftPos(13*pixelMagnifier, 13*pixelMagnifier)
	}

	sv := oakx.NewSignalVolumeSprite(0, 0, 18*pixelMagnifier, 18*pixelMagnifier)
	sv.SetActions(sig.TotalActions)

	hoverCh := make(chan struct{})
	btn.New(ctx, btn.Layers(0, 0),
		btn.Width(18*pixelMagnifier),
		btn.Height(18*pixelMagnifier),
		btn.Renderable(sv),
		btn.Pos(copyButton.X()-20*pixelMagnifier, copyButton.Y()),
		btn.Binding(event.Enter, oakx.RelativePhaseCollisionEnter(ctx)),
		btn.Binding(mouse.Start, func(b *entities.Entity, _ *mouse.Event) event.Response {
			go func() {
				var tooltip *entities.Entity
				select {
				case <-hoverCh:
					return
				case <-ctx.Done():
					return
				case <-time.After(600 * time.Millisecond):
					ev := mouse.LastEvent
					tooltip = btn.New(ctx, btn.Layers(1, 2),
						btn.Pos(ev.X(), ev.Y()-25*pixelMagnifier), // - 35 to render above the mouse itself
						btn.Width(70*pixelMagnifier),
						btn.Height(20*pixelMagnifier),
						btn.Font(fonts.get(fontNameTooltip)),
						btn.Color(color.RGBA{90, 90, 90, 100}),
						btn.Text("Actions: "+strconv.Itoa(sig.TotalActions)),
					)
				}
				<-hoverCh
				for _, child := range tooltip.Children {
					child.Destroy()
				}
				tooltip.Destroy()
			}()
			return 0
		}),
		btn.Binding(mouse.Stop, func(b *entities.Entity, _ *mouse.Event) event.Response {
			select {
			case hoverCh <- struct{}{}:
			default:
			}
			return 0
		}),
	)
	nextYStart += 45 * pixelMagnifier
	return nextYStart
}

func renderPageTitle(ctx *scene.Context, str string, x, y, width float64) *render.Sprite {
	title := fonts.get(fontNameSignature).NewText(str, x+30*pixelMagnifier, y+10*pixelMagnifier)
	//nolint:errcheck
	ctx.Draw(title, 0, 1)

	x1 := x - 10*pixelMagnifier
	x2 := x1 + width
	titleSep := render.NewColoredLine(x1, title.Y()+30*pixelMagnifier, x2, title.Y()+30*pixelMagnifier, render.IdentityColorer(colors.OffOffWhite), 1)
	//nolint:errcheck
	ctx.Draw(titleSep, 0, 2)
	return titleSep
}

func buildQRCode(st *state.State) (*render.Sprite, func()) {
	var qrCodeSprite *render.Sprite
	updateQRSprite := func() {
		img, err := st.QRCode()
		if err != nil {
			// TODO: this will always happen once the QR code is above an unreasonable nubmer of characters (how many?)
			fmt.Println("failed to create QR code", err)
			return
		}
		rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
		draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
		tmp := render.NewSprite(0, 0, rgba)
		tmpRGBA := tmp.Modify(mod.ResizeToFit(380, 380, mod.NearestNeighborResampling)).GetRGBA()
		if qrCodeSprite == nil {
			qrCodeSprite = render.NewSprite(0, 0, tmpRGBA)
		} else {
			qrCodeSprite.SetRGBA(tmpRGBA)
		}
	}
	updateQRSprite()
	return qrCodeSprite, updateQRSprite
}

func initTitlebar(ctx *scene.Context, titleBarHeight float64, darkMode bool) {
	event.GlobalBind(ctx, titlebar.WindowClosingEvent, func(struct{}) event.Response {
		if !keylogx.BackgroundWorkerAllowed {
			internal.GlobalMonitor.Stop()
		}
		return 0
	})
	titlebar.New(ctx, func(c titlebar.Constructor) titlebar.Constructor {
		c.Buttons = []titlebar.Button{titlebar.ButtonMinimize, titlebar.ButtonClose}
		c.ButtonWidth = titleBarHeight
		c.ButtonStyle = keylogx.ButtonStyle
		c.Height = titleBarHeight
		c.Color = Iff(darkMode, colors.Black, colors.GreenBlack)
		c.Layers = []int{1, 0}
		return c
	})
}

func initLogo(ctx *scene.Context, darkMode bool) {
	affiroLogoPath := path.Join("images", "affiro-fullmark-ondark-771x200.png")
	fonts.darkMode = darkMode

	x := 14 * pixelMagnifier
	y := 11 * pixelMagnifier
	if keylogx.ButtonStyle == titlebar.ButtonStyleOSX {
		x += 40 * pixelMagnifier * 2
	}

	const logoWidth = 65 * pixelMagnifier
	const logoHeight = 20 * pixelMagnifier
	affiroLogo, err := render.LoadSprite(affiroLogoPath)
	if err != nil {
		panic(err)
	}
	affiroLogo.Modify(mod.ResizeToFit(logoWidth, logoHeight, mod.LanczosResampling))
	affiroLogo.SetPos(x, y)
	//nolint:errcheck
	ctx.Draw(affiroLogo, 1, 1)
}

func initVersionText(ctx *scene.Context) {
	x := 88 * pixelMagnifier
	y := 12 * pixelMagnifier
	if keylogx.ButtonStyle == titlebar.ButtonStyleOSX {
		x += 40 * pixelMagnifier * 2
	}
	versionText := fonts.get(fontNameVersion).NewText(buildinfo.Version, x, y)
	//nolint:errcheck
	ctx.Draw(versionText, 1, 1)
}

func Iff[T any](darkMode bool, a, b T) T {
	if darkMode {
		return a
	}
	return b
}
