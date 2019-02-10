package gfx

import "github.com/go-gl/gl/v4.1-core/gl"

// Program is a wrapper to the gl program
type Program struct {
	// Handle is the program handle used in various gl_* calls
	Handle uint32
	// Shaders are a reference to a shader object,
	// used only for proper clean-up of resources.
	Shaders []*Shader
}

// MustMakeProgram instantiates a new shader program
// consisting of a vertex and fragment shader.
// This func panics on error
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

// NewProgram attaches the given shaders to the program
// and links it.
func NewProgram(shaders ...*Shader) (*Program, error) {
	prog := &Program{Handle: gl.CreateProgram()}
	prog.Attach(shaders...)

	if err := prog.Link(); err != nil {
		return nil, err
	}

	return prog, nil
}

// Delete deletes the program and all attached shaders
func (prog *Program) Delete() {
	for _, shader := range prog.Shaders {
		shader.Delete()
	}
	gl.DeleteProgram(prog.Handle)
}

// Attach attaches all given shaders to the program
func (prog *Program) Attach(shaders ...*Shader) {
	for _, shader := range shaders {
		gl.AttachShader(prog.Handle, shader.Handle)
		prog.Shaders = append(prog.Shaders, shader)
	}
}

// Use enables the program
func (prog *Program) Use() {
	gl.UseProgram(prog.Handle)
}

// Link links the program and checks the link status of the program
func (prog *Program) Link() error {
	gl.LinkProgram(prog.Handle)
	return getGlError(prog.Handle, gl.LINK_STATUS, gl.GetProgramiv, gl.GetProgramInfoLog,
		"program link failure")
}

// GetUniformLocation returns the location of a uniform variable
func (prog *Program) GetUniformLocation(name string) int32 {
	return gl.GetUniformLocation(prog.Handle, gl.Str(name+"\x00"))
}
