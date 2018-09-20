package gfx

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Shader -
type Shader struct {
	Handle uint32
}

// Program -
type Program struct {
	Handle  uint32
	Shaders []*Shader
}

// MustMakeProgram -
func MustMakeProgram(vert, frag string) *Program {
	vertShader, err := NewShader(vert, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragShader, err := NewShader(frag, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}
	shaderProgram, err := NewProgram(vertShader, fragShader)
	if err != nil {
		panic(err)
	}
	return shaderProgram
}

// Delete -
func (shader *Shader) Delete() {
	gl.DeleteShader(shader.Handle)
}

// Delete -
func (prog *Program) Delete() {
	for _, shader := range prog.Shaders {
		shader.Delete()
	}
	gl.DeleteProgram(prog.Handle)
}

func (prog *Program) Attach(shaders ...*Shader) {
	for _, shader := range shaders {
		gl.AttachShader(prog.Handle, shader.Handle)
		prog.Shaders = append(prog.Shaders, shader)
	}
}

func (prog *Program) Use() {
	gl.UseProgram(prog.Handle)
}

func (prog *Program) Link() error {
	gl.LinkProgram(prog.Handle)
	return getGlError(prog.Handle, gl.LINK_STATUS, gl.GetProgramiv, gl.GetProgramInfoLog,
		"program link failure")
}

func (prog *Program) GetUniformLocation(name string) int32 {
	return gl.GetUniformLocation(prog.Handle, gl.Str(name+"\x00"))
}

func NewProgram(shaders ...*Shader) (*Program, error) {
	prog := &Program{Handle: gl.CreateProgram()}
	prog.Attach(shaders...)

	if err := prog.Link(); err != nil {
		return nil, err
	}

	return prog, nil
}

func NewShader(src string, sType uint32) (*Shader, error) {

	handle := gl.CreateShader(sType)
	glSrc, freeFn := gl.Strs(src + "\x00")
	defer freeFn()
	gl.ShaderSource(handle, 1, glSrc, nil)
	gl.CompileShader(handle)
	err := getGlError(handle, gl.COMPILE_STATUS, gl.GetShaderiv, gl.GetShaderInfoLog,
		"shader compile failure")
	if err != nil {
		return nil, err
	}
	return &Shader{Handle: handle}, nil
}

func NewShaderFromFile(file string, sType uint32) (*Shader, error) {
	src, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	handle := gl.CreateShader(sType)
	glSrc, freeFn := gl.Strs(string(src) + "\x00")
	defer freeFn()
	gl.ShaderSource(handle, 1, glSrc, nil)
	gl.CompileShader(handle)
	err = getGlError(handle, gl.COMPILE_STATUS, gl.GetShaderiv, gl.GetShaderInfoLog,
		"shader compile failure"+file)
	if err != nil {
		return nil, err
	}
	return &Shader{Handle: handle}, nil
}

type getObjIv func(uint32, uint32, *int32)
type getObjInfoLog func(uint32, int32, *int32, *uint8)

func getGlError(glHandle uint32, checkTrueParam uint32, getObjIvFn getObjIv,
	getObjInfoLogFn getObjInfoLog, failMsg string) error {

	var success int32
	getObjIvFn(glHandle, checkTrueParam, &success)

	if success == gl.FALSE {
		var logLength int32
		getObjIvFn(glHandle, gl.INFO_LOG_LENGTH, &logLength)

		log := gl.Str(strings.Repeat("\x00", int(logLength)))
		getObjInfoLogFn(glHandle, logLength, nil, log)

		return fmt.Errorf("%s: %s", failMsg, gl.GoStr(log))
	}

	return nil
}
