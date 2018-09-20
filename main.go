package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/keybind"
	"github.com/BurntSushi/xgbutil/mousebind"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xwindow"
	"github.com/gobuffalo/packr"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.1/glfw"

	"github.com/moolen/glitchlock/pam"
	"github.com/moolen/gllock/gfx"
	"github.com/moolen/gllock/gfx/gvd"
	log "github.com/sirupsen/logrus"
)

func init() {
	runtime.LockOSThread()
}

var maxFPS = 60.0
var maxTime = 1.0 / maxFPS

var version = "dev"

func main() {

	flagVersion := flag.Bool("version", false, "show version and exit")
	flagOverlay := flag.String("overlay", "", "specify a path to an image. it will be overlayed at the center of the screen")
	flagBackground := flag.String("bg", "", "specify a path to an image. it will be the background of the screen")
	flagDebug := flag.Bool("debug", false, "debug mode: additional log info")
	flag.Parse()

	if *flagVersion {
		fmt.Printf("gllock %s\n", version)
		return
	}

	if *flagBackground == "" {
		fmt.Println("no background specifed but required")
		return
	}

	if *flagDebug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("enabled debug mode")
	}

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to inifitialize glfw:", err)
	}
	defer glfw.Terminate()

	primaryMonitor := glfw.GetPrimaryMonitor()
	videoMode := primaryMonitor.GetVideoMode()
	mons := glfw.GetMonitors()
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(videoMode.Width, videoMode.Height, "gllock", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		panic(err)
	}

	xorg, err := NewXorg()
	xorg.Fullscreen("gllock")
	err = xorg.GrabInput()
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			window.SetShouldClose(true)
			return
		}
	}()

	go func() {
		done := xorg.PasswordMatch()
		for {
			select {
			case <-done:
				window.SetShouldClose(true)
				return
			}
		}
	}()

	for _, monitor := range mons {
		x, y := monitor.GetPos()
		if monitor.GetName() != primaryMonitor.GetName() {
			videoMode := monitor.GetVideoMode()
			log.Debugf("mon %#v %#v pos: %d, %d", monitor, primaryMonitor, x, y)
			xorg.Overlay(x, y, videoMode.Width, videoMode.Height)
		}
	}

	err = programLoop(window, *flagBackground, *flagOverlay, *videoMode)
	if err != nil {
		log.Fatal(err)
	}
}

func programLoop(window *glfw.Window, background, overlay string, videoMode glfw.VidMode) error {

	var overlayPlane *gfx.Mesh
	var overlayTex *gfx.Texture
	if overlay != "" {
		overlayTex = gfx.MustTextureFromFile(overlay, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
		overlayPlane = gfx.NewMesh(gvd.PlaneVertices, gvd.PlaneIndices, []*gfx.Texture{overlayTex})
	}
	backgroundFile, err := os.Open(background)
	if err != nil {
		return err
	}
	screen, _, err := image.Decode(backgroundFile)
	if err != nil {
		return err
	}

	box := packr.NewBox("shaders")
	screenTex := gfx.MustTexture(screen, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	planeProg := gfx.MustMakeProgram(box.String("regular.vert"), box.String("regular.frag"))
	screenshotPlane := gfx.NewMesh(gvd.PlaneVertices, gvd.PlaneIndices, []*gfx.Texture{screenTex})

	fbo := gfx.MustFramebuffer(videoMode.Width, videoMode.Height)
	defer fbo.Destroy()
	fxProg := gfx.MustMakeProgram(box.String("fx.vert"), box.String("fx.frag"))
	fxPlane := gfx.NewMesh(gvd.InvertedTexPlaneVertices, gvd.PlaneIndices, []*gfx.Texture{fbo.Texture})

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)

	var time, delta, lastTime float64
	time = glfw.GetTime()

	fxProg.Use()
	gl.Uniform2i(fxProg.GetUniformLocation("resolution"), int32(videoMode.Height), int32(videoMode.Width))

	for !window.ShouldClose() {
		glfw.PollEvents()
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		time = glfw.GetTime()
		delta = time - lastTime

		if delta < maxTime {
			continue
		}

		lastTime = time

		// render to framebuffer
		fbo.Bind()
		planeProg.Use()
		screenshotPlane.Draw(planeProg)
		fbo.Unbind()

		// render framebuffer to screen
		fxProg.Use()
		gl.Uniform1f(fxProg.GetUniformLocation("time"), float32(glfw.GetTime()))
		fxPlane.Draw(fxProg)

		if overlayTex != nil && overlayPlane != nil {
			// render archlinux image to screen
			planeProg.Use()
			gl.Viewport(int32(videoMode.Width/2)-overlayTex.Width/2, int32(videoMode.Height/2)-overlayTex.Height/2, overlayTex.Width, overlayTex.Height)
			overlayPlane.Draw(planeProg)
			gl.Viewport(0, 0, int32(videoMode.Width), int32(videoMode.Height))
		}

		window.SwapBuffers()
	}
	return nil
}

type Xorg struct {
	X  *xgb.Conn
	Xu *xgbutil.XUtil
}

func NewXorg() (*Xorg, error) {
	X, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	Xu, err := xgbutil.NewConnXgb(X)
	if err != nil {
		return nil, err
	}
	return &Xorg{X, Xu}, nil
}

func (x *Xorg) GrabInput() error {
	keybind.Initialize(x.Xu)
	xscreen := xproto.Setup(x.X).DefaultScreen(x.X)
	grabc := xproto.GrabKeyboard(x.X, false, xscreen.Root, xproto.TimeCurrentTime,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
	)
	repk, err := grabc.Reply()
	if err != nil {
		return fmt.Errorf("error grabbing Keyboard")
	}
	if repk.Status != xproto.GrabStatusSuccess {
		return fmt.Errorf("could not grab keyboard")
	}
	grabp := xproto.GrabPointer(x.X, false, xscreen.Root, (xproto.EventMaskKeyPress|xproto.EventMaskKeyRelease)&0,
		xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.CursorNone, xproto.TimeCurrentTime)
	repp, err := grabp.Reply()
	if err != nil {
		return fmt.Errorf("error grabbing pointer")
	}
	if repp.Status != xproto.GrabStatusSuccess {
		return fmt.Errorf("could not grab pointer")
	}
	return nil
}

func (x *Xorg) FindWindow(name string) (xproto.Window, error) {
	clientids, err := ewmh.ClientListGet(x.Xu)
	if err != nil {
		return 0, err
	}
	for _, clientid := range clientids {
		winName, err := ewmh.WmNameGet(x.Xu, clientid)
		if err != nil || len(winName) == 0 {
			winName, err = icccm.WmNameGet(x.Xu, clientid)
			if err != nil || len(winName) == 0 {
				winName = "N/A"
			}
		}
		if winName == name {
			return clientid, nil
		}
	}
	return 0, fmt.Errorf("X Window not found")
}

func (x *Xorg) Fullscreen(name string) error {
	win, err := x.FindWindow(name)
	if err != nil {
		return err
	}
	err = ewmh.WmStateReq(x.Xu, win, ewmh.StateToggle,
		"_NET_WM_STATE_FULLSCREEN")
	if err != nil {
		return err
	}
	return nil
}

func (x *Xorg) PasswordMatch() <-chan struct{} {
	lastInput := time.Now()
	done := make(chan struct{}, 1)

	go func() {
		var password string
		for {
			ev, err := x.X.WaitForEvent()
			if ev == nil && err == nil {
				log.Error(fmt.Errorf("Both event and error are nil. Exiting"))
				done <- struct{}{}
				return
			}
			if err != nil {
				log.Error(fmt.Errorf("Both event and error are nil. Exiting"))
				done <- struct{}{}
				return
			}
			if time.Now().Sub(lastInput) > time.Second*2 {
				log.Debugf("timeout reached. clearing password")
				password = ""
			}
			switch e := ev.(type) {
			case xproto.KeyPressEvent:
				key := keybind.LookupString(x.Xu, e.State, e.Detail)
				log.Debugf("keypress: %s %v ", key, e)
				lastInput = time.Now()
				if len(key) == 1 {
					password += key
				}
				if keybind.KeyMatch(x.Xu, "BackSpace", e.State, e.Detail) && len(password) > 0 {
					password = password[:len(password)-1]
				}
				log.Debugf("current password: %s", password)
				if keybind.KeyMatch(x.Xu, "Return", e.State, e.Detail) {
					log.Debugf("...checking password")
					if pam.AuthenticateCurrentUser(password) {
						done <- struct{}{}
						return
					}
					log.Debugf("password does not match")
				}
			}
		}
	}()
	return done
}

func (xorg *Xorg) Overlay(x, y, width, height int) error {
	log.Debugf("overlay size: %d %d %d %d", x, y, x+width, y+height)
	rgba := image.NewRGBA(image.Rect(x, y, x+width, y+height))
	ximg := xgraphics.NewConvert(xorg.Xu, rgba)
	win, err := xwindow.Generate(xorg.Xu)
	if err != nil {
		return err
	}
	win.Create(xorg.Xu.RootWin(), x, y, width, height, 0)
	win.WMGracefulClose(func(w *xwindow.Window) {
		xevent.Detach(w.X, w.Id)
		keybind.Detach(w.X, w.Id)
		mousebind.Detach(w.X, w.Id)
		w.Destroy()
	})
	err = icccm.WmStateSet(xorg.Xu, win.Id, &icccm.WmState{
		State: icccm.StateNormal,
	})
	if err != nil {
		return err
	}
	err = icccm.WmNormalHintsSet(xorg.Xu, win.Id, &icccm.NormalHints{
		Flags:     icccm.SizeHintPMinSize | icccm.SizeHintPMaxSize,
		MinWidth:  uint(width),
		MinHeight: uint(height),
		MaxWidth:  uint(width),
		MaxHeight: uint(height),
	})
	if err != nil {
		return err
	}

	err = ewmh.WmStateReq(xorg.Xu, win.Id, ewmh.StateToggle,
		"_NET_WM_STATE_FULLSCREEN")
	if err != nil {
		return err
	}

	// Paint our image before mapping.
	ximg.XSurfaceSet(win.Id)
	ximg.XDraw()
	ximg.XPaint(win.Id)
	win.Map()

	// some WM override this position after mapping
	win.Move(x, y)
	return nil
}
