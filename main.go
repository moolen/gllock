package main

import (
	"log"
	"runtime"

	"github.com/kbinani/screenshot"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.1/glfw"

	"github.com/moolen/gllock/gfx"
)

func init() {
	runtime.LockOSThread()
}

func main() {
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
	window, err := glfw.CreateWindow(videoMode.Width, videoMode.Height, "basic textures", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		panic(err)
	}

	window.SetKeyCallback(keyCallback)

	err = programLoop(window, *videoMode)
	if err != nil {
		log.Fatal(err)
	}
}

func programLoop(window *glfw.Window, videoMode glfw.VidMode) error {
	screen, err := screenshot.CaptureDisplay(0)
	if err != nil {
		panic(err)
	}
	texture0, err := gfx.NewTexture(screen,
		gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	if err != nil {
		panic(err.Error())
	}

	planeProg, err := makeProgram("shaders/regular.vert", "shaders/regular.frag")
	if err != nil {
		panic(err)
	}
	plane, err := gfx.NewTexturedPlane(planeVertices, planeIndices, texture0, planeProg)
	if err != nil {
		return err
	}
	defer plane.Delete()

	// todo: abstract FBO binding / render-to-texture
	var FramebufferName uint32
	gl.GenFramebuffers(1, &FramebufferName)
	gl.BindFramebuffer(gl.FRAMEBUFFER, FramebufferName)

	var renderedTexture uint32
	gl.GenTextures(1, &renderedTexture)
	gl.BindTexture(gl.TEXTURE_2D, renderedTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, int32(videoMode.Width), int32(videoMode.Height), 0, gl.RGB, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	var depthrenderbuffer uint32
	gl.GenRenderbuffers(1, &depthrenderbuffer)
	gl.BindRenderbuffer(gl.RENDERBUFFER, depthrenderbuffer)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT, int32(videoMode.Width), int32(videoMode.Height))
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthrenderbuffer)

	gl.FramebufferTexture(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, renderedTexture, 0)
	DrawBuffers := uint32(gl.COLOR_ATTACHMENT0)
	gl.DrawBuffers(1, &DrawBuffers)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		panic("FBO status broken")
	}

	fboTex := gfx.NewRawTexture(renderedTexture, gl.TEXTURE_2D, gl.TEXTURE0)

	fxProg, err := makeProgram("shaders/fx.vert", "shaders/fx.frag")
	if err != nil {
		panic(err)
	}
	fxPlane, err := gfx.NewTexturedPlane(invertedTexPlaneVertices, planeIndices, fboTex, fxProg)
	if err != nil {
		panic(err)
	}

	for !window.ShouldClose() {
		glfw.PollEvents()
		fxPlane.Time = glfw.GetTime()
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.BindFramebuffer(gl.FRAMEBUFFER, FramebufferName)
		plane.Draw()
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

		fxPlane.Draw()

		window.SwapBuffers()
	}

	gl.DeleteFramebuffers(1, &FramebufferName)

	return nil
}

func makeProgram(vert, frag string) (*gfx.Program, error) {
	vertShader, err := gfx.NewShaderFromFile(vert, gl.VERTEX_SHADER)
	if err != nil {
		return nil, err
	}
	fragShader, err := gfx.NewShaderFromFile(frag, gl.FRAGMENT_SHADER)
	if err != nil {
		return nil, err
	}
	shaderProgram, err := gfx.NewProgram(vertShader, fragShader)
	if err != nil {
		return nil, err
	}
	return shaderProgram, nil
}

func keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action,
	mods glfw.ModifierKey) {
	if key == glfw.KeyEscape && action == glfw.Press {
		window.SetShouldClose(true)
	}
}

var planeVertices = []float32{
	// top left
	-1.0, 1.0, 0.0, // position
	0.0, 0.0, // texture coordinates

	// top right
	1.0, 1.0, 0.0,
	1.0, 0.0,

	// bottom right
	1.0, -1.0, 0.0,
	1.0, 1.0,

	// bottom left
	-1.0, -1.0, 0.0,
	0.0, 1.0,
}

var invertedTexPlaneVertices = []float32{
	// top left
	-1.0, 1.0, 0.0, // position
	0.0, 1.0, // texture coordinates

	// top right
	1.0, 1.0, 0.0,
	1.0, 1.0,

	// bottom right
	1.0, -1.0, 0.0,
	1.0, 0.0,

	// bottom left
	-1.0, -1.0, 0.0,
	0.0, 0.0,
}

var planeIndices = []uint32{
	// rectangle
	0, 1, 2, // top triangle
	0, 2, 3, // bottom triangle
}
