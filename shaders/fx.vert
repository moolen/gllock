#version 410 core

layout (location = 0) in vec3 position;
layout (location = 1) in vec3 normal;
layout (location = 2) in vec2 texCoord;

out vec2 TexCoord;
out vec3 vec_position;

void main()
{
    gl_Position = vec4(position, 1.0);
    TexCoord = texCoord;
    vec_position = position;
}
