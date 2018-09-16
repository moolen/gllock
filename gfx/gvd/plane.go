package gvd

var PlaneVertices = []float32{
	// top left
	-1.0, 1.0, 0.0, // position
	0, 0, 0, // normal
	0.0, 0.0, // texture coordinates

	// top right
	1.0, 1.0, 0.0,
	0, 0, 0,
	1.0, 0.0,

	// bottom right
	1.0, -1.0, 0.0,
	0, 0, 0,
	1.0, 1.0,

	// bottom left
	-1.0, -1.0, 0.0,
	0, 0, 0,
	0.0, 1.0,
}

var InvertedTexPlaneVertices = []float32{
	// top left
	-1.0, 1.0, 0.0, // position
	0, 0, 0, // normal
	0.0, 1.0, // texture coordinates

	// top right
	1.0, 1.0, 0.0,
	0, 0, 0,
	1.0, 1.0,

	// bottom right
	1.0, -1.0, 0.0,
	0, 0, 0,
	1.0, 0.0,

	// bottom left
	-1.0, -1.0, 0.0,
	0, 0, 0,
	0.0, 0.0,
}

var PlaneIndices = []uint32{
	// rectangle
	0, 1, 2, // top triangle
	0, 2, 3, // bottom triangle
}
