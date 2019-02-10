package gfx

import (
	"fmt"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Mesh is a collection of vertices, that
// define the shape of a 3D object
type Mesh struct {
	// VAO is the vertex array object that stores the vertex data
	VAO uint32
	// VBO is the vertex buffer object used as a source for vertex array data
	VBO uint32
	// EBO is the element buffer object used to store the indices for the vertex data
	EBO uint32
	// Vertices are the actual points - used only for the initialization
	Vertices []float32
	// Indices are the actual indices for the vertices - used only for the initialization
	Indices []uint32
	// Textures are the textures that the Mesh references in the fragment shader
	Textures []*Texture
}

// NewMesh allocates the buffer objects
func NewMesh(vertices []float32, indices []uint32, textures []*Texture) *Mesh {
	VAO, VBO, EBO := createVAO(vertices, indices)
	return &Mesh{
		VAO:      VAO,
		VBO:      VBO,
		EBO:      EBO,
		Vertices: vertices,
		Indices:  indices,
		Textures: textures,
	}
}

// Draw binds all referenced textures and vertex array objects
// and draws triangles
func (m *Mesh) Draw(program *Program) {
	for i, tex := range m.Textures {
		tex.Bind(uint32(gl.TEXTURE0 + i))
		location := program.GetUniformLocation(fmt.Sprintf("texture%d", i))
		gl.Uniform1i(location, int32(i))
	}
	gl.BindVertexArray(m.VAO)
	gl.DrawElements(gl.TRIANGLES, int32(len(m.Indices)), gl.UNSIGNED_INT, unsafe.Pointer(nil))
	gl.BindVertexArray(0)

	for _, tex := range m.Textures {
		tex.Unbind()
	}
}

func createVAO(vertices []float32, indices []uint32) (uint32, uint32, uint32) {
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
	var stride int32 = 3*4 + 3*4 + 2*4
	var offset int

	// position
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(offset))
	gl.EnableVertexAttribArray(0)
	offset += 3 * 4

	// position
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(offset))
	gl.EnableVertexAttribArray(1)
	offset += 3 * 4

	// texture
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, stride, gl.PtrOffset(offset))
	gl.EnableVertexAttribArray(2)
	offset += 2 * 4

	gl.BindVertexArray(0)
	return VAO, VBO, EBO
}
