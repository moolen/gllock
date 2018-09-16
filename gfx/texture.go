package gfx

import (
	"errors"
	"image"
	"image/draw"

	// decode images
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Texture -
type Texture struct {
	Handle uint32
	Width  int32
	Height int32
}

var errUnsupportedStride = errors.New("unsupported stride, only 32-bit colors supported")

// MustTexture -
func MustTexture(img image.Image, wrapR, wrapS int32) *Texture {
	tex, err := NewTexture(img, wrapR, wrapS)
	if err != nil {
		panic(err)
	}
	return tex
}

func NewTexture(img image.Image, wrapR, wrapS int32) (*Texture, error) {
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Pt(0, 0), draw.Src)
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, errUnsupportedStride
	}
	width := int32(rgba.Rect.Size().X)
	height := int32(rgba.Rect.Size().Y)
	texture := Texture{
		Width:  width,
		Height: height,
	}
	gl.GenTextures(1, &texture.Handle)

	texture.Bind(gl.TEXTURE0)
	defer texture.Unbind()

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_R, wrapR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, wrapS)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	gl.TexImage2D(gl.TEXTURE_2D, 0, int32(gl.SRGB_ALPHA), texture.Width, texture.Height, 0, uint32(gl.RGBA), uint32(gl.UNSIGNED_BYTE), gl.Ptr(rgba.Pix))

	gl.GenerateMipmap(texture.Handle)

	return &texture, nil
}

// Bind -
func (tex *Texture) Bind(unit uint32) {
	gl.ActiveTexture(unit)
	gl.BindTexture(gl.TEXTURE_2D, tex.Handle)
}

// Unbind -
func (tex *Texture) Unbind() {
	gl.BindTexture(gl.TEXTURE_2D, 0)
}

func MustTextureFromFile(path string, wrapR, wrapS int32) *Texture {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	return MustTexture(img, wrapR, wrapS)
}
