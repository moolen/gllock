#version 410 core

in vec2 TexCoord;

layout(location = 0) out vec4 color;

uniform sampler2D texture0;

void main()
{
    color = texture(texture0, TexCoord);
}
