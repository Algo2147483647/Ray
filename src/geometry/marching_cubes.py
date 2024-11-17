import numpy as np

# Marching Cubes lookup table (partially filled for brevity)
MarchingCubes_TriTable = [
    [-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1],
    [0, 8, 3, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1],
    [0, 1, 9, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1]
    # Complete the table with other values as necessary
]


def marching_cubes(f, st, ed, N):
    vertex = [0, 1, 3, 2, 4, 5, 7, 6]
    edge = [
        [0, 1], [1, 2], [3, 2], [0, 3],
        [4, 5], [5, 6], [7, 6], [4, 7],
        [0, 4], [1, 5], [2, 6], [3, 7]
    ]
    triangle_set = []
    delta = [(ed[0] - st[0]) / N[0], (ed[1] - st[1]) / N[1], (ed[2] - st[2]) / N[2]]

    for i in range(N[0] * N[1] * N[2]):
        vec = [
            st[0] + (i % N[0]) * delta[0],
            st[1] + ((i // N[0]) % N[1]) * delta[1],
            st[2] + (i // (N[0] * N[1])) * delta[2]
        ]

        cubeindex = 0
        val = [0] * 8

        for j in range(8):
            x = vec[0] + delta[0] * (vertex[j] & 0b1)
            y = vec[1] + delta[1] * ((vertex[j] & 0b10) >> 1)
            z = vec[2] + delta[2] * ((vertex[j] & 0b100) >> 2)

            if x < st[0] or x > ed[0] or y < st[1] or y > ed[1] or z < st[2] or z > ed[2]:
                continue

            val[j] = f(x, y, z)
            if val[j] < 0:
                cubeindex |= (1 << j)

        for j in range(0, 15, 3):
            if MarchingCubes_TriTable[cubeindex][j] == -1:
                break

            tri = [0] * 9
            for k in range(3):
                e = MarchingCubes_TriTable[cubeindex][j + k]
                a = -val[edge[e][0]] / (val[edge[e][1]] - val[edge[e][0]])
                dx = (vertex[edge[e][0]] & 0b1) * (1 - a) + (vertex[edge[e][1]] & 0b1) * a
                dy = ((vertex[edge[e][0]] & 0b10) >> 1) * (1 - a) + ((vertex[edge[e][1]] & 0b10) >> 1) * a
                dz = ((vertex[edge[e][0]] & 0b100) >> 2) * (1 - a) + ((vertex[edge[e][1]] & 0b100) >> 2) * a

                tri[k * 3 + 0] = vec[0] + delta[0] * dx
                tri[k * 3 + 1] = vec[1] + delta[1] * dy
                tri[k * 3 + 2] = vec[2] + delta[2] * dz

            triangle_set.append(tri)

    return triangle_set


def marching_cubes_grid(X, zero, delta):
    vertex = [0, 1, 3, 2, 4, 5, 7, 6]
    edge = [
        [0, 1], [1, 2], [3, 2], [0, 3],
        [4, 5], [5, 6], [7, 6], [4, 7],
        [0, 4], [1, 5], [2, 6], [3, 7]
    ]
    triangle_set = []
    N = [len(X[0][0]), len(X[0]), len(X)]

    for i in range(N[0] * N[1] * N[2]):
        cubeindex = 0
        val = [0] * 8

        for j in range(8):
            x = (i % N[0]) + (vertex[j] & 0b1)
            y = ((i // N[0]) % N[1]) + ((vertex[j] & 0b10) >> 1)
            z = (i // (N[0] * N[1])) + ((vertex[j] & 0b100) >> 2)

            if x < 0 or x >= N[0] or y < 0 or y >= N[1] or z < 0 or z >= N[2]:
                continue

            val[j] = X[z][y][x]
            if val[j] <= 0:
                cubeindex |= (1 << j)

        for j in range(0, 15, 3):
            if MarchingCubes_TriTable[cubeindex][j] == -1:
                break

            tri = [0] * 9
            for k in range(3):
                e = MarchingCubes_TriTable[cubeindex][j + k]
                a = -val[edge[e][0]] / (val[edge[e][1]] - val[edge[e][0]])
                dx = (vertex[edge[e][0]] & 0b1) * (1 - a) + (vertex[edge[e][1]] & 0b1) * a
                dy = ((vertex[edge[e][0]] & 0b10) >> 1) * (1 - a) + ((vertex[edge[e][1]] & 0b10) >> 1) * a
                dz = ((vertex[edge[e][0]] & 0b100) >> 2) * (1 - a) + ((vertex[edge[e][1]] & 0b100) >> 2) * a

                tri[k * 3 + 0] = zero[0] + ((i % N[0]) + dx) * delta[0]
                tri[k * 3 + 1] = zero[1] + (((i // N[0]) % N[1]) + dy) * delta[1]
                tri[k * 3 + 2] = zero[2] + ((i // (N[0] * N[1])) + dz) * delta[2]

            triangle_set.append(tri)

    return triangle_set


# Example usage
def test_function(x, y, z):
    return x ** 2 + y ** 2 + z ** 2 - 1


st = [0.0, 0.0, 0.0]
ed = [1.0, 1.0, 1.0]
N = [10, 10, 10]

triangle_set = marching_cubes(test_function, st, ed, N)
print("Triangles:", triangle_set)
