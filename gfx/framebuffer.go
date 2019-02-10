package gfx

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

// Framebuffer is a wrapper for the gl framebuffer
type Framebuffer struct {
	Handle        uint32
	RenderTexture uint32
	DepthBuffer   uint32
	Texture       *Texture
}

// MustFramebuffer -
func MustFramebuffer(width, height int) *Framebuffer {
	framebuffer, err := NewFramebuffer(width, height)
	if err != nil {
		panic(err)
	}
	return framebuffer
}

func NewFramebuffer(width, height int) (*Framebuffer, error) {
	var framebufferHandle uint32
	gl.GenFramebuffers(1, &framebufferHandle)
	gl.BindFramebuffer(gl.FRAMEBUFFER, framebufferHandle)

	var renderTexture uint32
	gl.GenTextures(1, &renderTexture)
	gl.BindTexture(gl.TEXTURE_2D, renderTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, int32(width), int32(height), 0, gl.RGB, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	var depthbuffer uint32
	gl.GenRenderbuffers(1, &depthbuffer)
	gl.BindRenderbuffer(gl.RENDERBUFFER, depthbuffer)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT, int32(width), int32(height))
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthbuffer)

	gl.FramebufferTexture(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, renderTexture, 0)
	DrawBuffers := uint32(gl.COLOR_ATTACHMENT0)
	gl.DrawBuffers(1, &DrawBuffers)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		panic("FBO status broken")
	}

	return &Framebuffer{
		Handle:        framebufferHandle,
		RenderTexture: renderTexture,
		DepthBuffer:   depthbuffer,
		Texture: &Texture{
			Handle: renderTexture,
			Width:  int32(width),
			Height: int32(height),
		},
	}, nil
}

func (f *Framebuffer) Bind() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, f.Handle)

}

func (f *Framebuffer) Unbind() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

}

// Destroy -
func (f *Framebuffer) Destroy() {
	gl.DeleteFramebuffers(1, &f.Handle)

}
