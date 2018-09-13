package gfx

import (
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

type TexturedPlane struct {
	VAO     uint32
	texture *Texture
	program *Program
	Time    float64
}

func NewTexturedPlane(vertices []float32, indices []uint32, texture *Texture, shaderProgram *Program) (*TexturedPlane, error) {
	VAO := createVAO(vertices, indices)
	return &TexturedPlane{
		VAO, texture, shaderProgram, 0,
	}, nil
}

// TODO: use Bind / Draw / Unbind semantics
//
func (m *TexturedPlane) Draw() {
	m.program.Use()

	m.texture.Bind(gl.TEXTURE0)
	m.texture.SetUniform(m.program.GetUniformLocation("texture0"))

	// refactor uniform binding
	gl.Uniform1f(m.program.GetUniformLocation("time"), float32(m.Time))

	gl.BindVertexArray(m.VAO)
	// todo: make variable
	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, unsafe.Pointer(nil))
	gl.BindVertexArray(0)

	m.texture.UnBind()
}

func (m *TexturedPlane) Delete() {
	m.program.Delete()
}

func createVAO(vertices []float32, indices []uint32) uint32 {

	var VAO, VBO, EBO uint32
	gl.GenVertexArrays(1, &VAO)
	gl.GenBuffers(1, &VBO)
	gl.GenBuffers(1, &EBO)

	gl.BindVertexArray(VAO)

	// vertices
	gl.BindBuffer(gl.ARRAY_BUFFER, VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// indices
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, EBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	// stride = sum of attributes
	var stride int32 = 3*4 + 2*4
	var offset int

	// position
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(offset))
	gl.EnableVertexAttribArray(0)
	offset += 3 * 4

	// texture
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(offset))
	gl.EnableVertexAttribArray(1)
	offset += 2 * 4

	// unbind VAO
	gl.BindVertexArray(0)

	return VAO
}
