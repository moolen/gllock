package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"os/signal"
	"runtime"

	"github.com/moolen/glitchlock/snap"

	"github.com/gobuffalo/packr"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.1/glfw"

	"github.com/moolen/gllock/gfx"
	"github.com/moolen/gllock/gfx/gvd"
	"github.com/moolen/gllock/xw"
	log "github.com/sirupsen/logrus"
)

func init() {
	runtime.LockOSThread()
}

var maxFPS = 24.0
var maxTime = 1.0 / maxFPS

var version = "dev"

func main() {

	flagVersion := flag.Bool("version", false, "show version and exit")
	flagOverlay := flag.String("overlay", "", "specify a path to an image. it will be overlayed at the center of the screen. This image should be smaller than the screen dimensions.")
	flagDebug := flag.Bool("debug", false, "debug mode logs additional information (for instance your password)")
	flag.Parse()

	if *flagVersion {
		fmt.Printf("gllock %s\n", version)
		return
	}

	if *flagDebug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("enabled debug mode")
	}

	// capture screen before we create the glfw window
	// (if we'd do it later we'd run into a race condition)
	primaryScreen, err := snap.GetPrimary()
	if err != nil {
		panic(err)
	}
	primaryScreenSnapshot, err := primaryScreen.Capture()
	if err != nil {
		panic(err)
	}

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to inifitialize glfw:", err)
	}
	defer glfw.Terminate()

	// setup monitor & window
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

	xw, err := xw.New()
	if err != nil {
		panic(err)
	}

	// race-condition? window might not yet be there?
	err = xw.Fullscreen("gllock")
	if err != nil {
		panic(err)
	}
	err = xw.GrabInput()
	if err != nil {
		log.Fatal(err)
	}

	// SIGINT handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Debugf("received SIGINT, closing window")
			window.SetShouldClose(true)
			return
		}
	}()

	// password-matcher goroutine
	go func() {
		done := xw.PasswordMatch()
		for {
			select {
			case <-done:
				window.SetShouldClose(true)
				return
			}
		}
	}()

	// overlay non-primary monitors with black
	for _, monitor := range mons {
		if monitor.GetName() != primaryMonitor.GetName() {
			x, y := monitor.GetPos()
			videoMode := monitor.GetVideoMode()
			log.Debugf("overlaying monitor %s, videoMode: %#v pos: %d, %d", monitor.GetName(), videoMode, x, y)
			xw.Overlay(x, y, videoMode.Width, videoMode.Height)
		}
	}

	// this runs until our glfw window receives a ShouldClose() call
	err = programLoop(window, primaryScreen, primaryScreenSnapshot, *flagOverlay, *videoMode)
	if err != nil {
		log.Fatal(err)
	}
}

func programLoop(window *glfw.Window, primaryScreen snap.Screen, snapshot *image.RGBA, overlay string, videoMode glfw.VidMode) error {
	var overlayPlane *gfx.Mesh
	var overlayTex *gfx.Texture
	if overlay != "" {
		overlayTex = gfx.MustTextureFromFile(overlay, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
		overlayPlane = gfx.NewMesh(gvd.PlaneVertices, gvd.PlaneIndices, []*gfx.Texture{overlayTex})
	}

	log.Debugf("%#v", primaryScreen)
	window.SetPos(primaryScreen.X, primaryScreen.Y)
	box := packr.NewBox("shaders")
	screenTex := gfx.MustTexture(snapshot, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
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
			// render image to screen
			planeProg.Use()
			gl.Viewport(int32(videoMode.Width/2)-overlayTex.Width/2, int32(videoMode.Height/2)-overlayTex.Height/2, overlayTex.Width, overlayTex.Height)
			overlayPlane.Draw(planeProg)
			gl.Viewport(0, 0, int32(videoMode.Width), int32(videoMode.Height))
		}

		window.SwapBuffers()
	}
	return nil
}
