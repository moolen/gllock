package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/kbinani/screenshot"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.1/glfw"

	"github.com/moolen/gllock/gfx"
	"github.com/moolen/gllock/gfx/gvd"
	log "github.com/sirupsen/logrus"
)

func init() {
	runtime.LockOSThread()
}

const maxFPS = 25.0
const maxTime = 1.0 / maxFPS

var version = "dev"

func main() {

	flagVersion := flag.Bool("version", false, "show version and exit")
	flagOverlay := flag.String("overlay", "", "overlay image")
	flag.Parse()

	if *flagVersion {
		fmt.Printf("gllock %s\n", version)
		return
	}

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to inifitialize glfw:", err)
	}
	defer glfw.Terminate()

	videoMode := glfw.GetPrimaryMonitor().GetVideoMode()
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

	window.SetKeyCallback(keyCallback)

	err = programLoop(window, *flagOverlay, *videoMode)
	if err != nil {
		log.Fatal(err)
	}
}

func programLoop(window *glfw.Window, overlay string, videoMode glfw.VidMode) error {
	screen, err := screenshot.CaptureDisplay(0)
	if err != nil {
		panic(err)
	}
	var overlayPlane *gfx.Mesh
	var overlayTex *gfx.Texture
	if overlay != "" {
		overlayTex = gfx.MustTextureFromFile(overlay, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
		overlayPlane = gfx.NewMesh(gvd.PlaneVertices, gvd.PlaneIndices, []*gfx.Texture{overlayTex})
	}

	screenTex := gfx.MustTexture(screen, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	planeProg := gfx.MustMakeProgram("shaders/regular.vert", "shaders/regular.frag")
	screenshotPlane := gfx.NewMesh(gvd.PlaneVertices, gvd.PlaneIndices, []*gfx.Texture{screenTex})

	fbo := gfx.MustFramebuffer(videoMode.Width, videoMode.Height)
	defer fbo.Destroy()
	fxProg := gfx.MustMakeProgram("shaders/fx.vert", "shaders/fx.frag")
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

func keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action,
	mods glfw.ModifierKey) {
	if key == glfw.KeyEscape && action == glfw.Press {
		window.SetShouldClose(true)
	}
}
