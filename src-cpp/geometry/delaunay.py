import numpy as np


def sort_points(points):
    return sorted(points, key=lambda p: (p[0], p[1]))


def subtract(a, b):
    return [a[0] - b[0], a[1] - b[1]]


def norm(v):
    return np.sqrt(v[0] ** 2 + v[1] ** 2)


def three_points_to_circle(tri_edge):
    # Calculate circumcenter and radius of the circle passing through three points
    ax, ay = tri_edge[0]
    bx, by = tri_edge[1]
    cx, cy = tri_edge[2]

    d = 2 * (ax * (by - cy) + bx * (cy - ay) + cx * (ay - by))
    if d == 0:
        return None, float('inf')  # Degenerate case

    ux = ((ax ** 2 + ay ** 2) * (by - cy) + (bx ** 2 + by ** 2) * (cy - ay) + (cx ** 2 + cy ** 2) * (ay - by)) / d
    uy = ((ax ** 2 + ay ** 2) * (cx - bx) + (bx ** 2 + by ** 2) * (ax - cx) + (cx ** 2 + cy ** 2) * (bx - ax)) / d
    center = [ux, uy]
    radius = norm(subtract(tri_edge[0], center))
    return center, radius


def delaunay(points):
    points = sort_points(points)

    tri_ans = []
    tri_temp = []
    edge_buffer = []

    min_point = points[0]
    max_point = points[-1]
    length = subtract(max_point, min_point)

    supertriangle = [
        [min_point[0] - length[0] - 2, min_point[1] - 2],
        [max_point[0] + length[0] + 2, min_point[1] - 2],
        [(max_point[0] + min_point[0]) / 2, max_point[1] + length[1] + 2],
    ]
    tri_temp.append(supertriangle)

    for point in points:
        edge_buffer.clear()

        for j in range(len(tri_temp) - 1, -1, -1):
            tri_edge = tri_temp[j]
            center, radius = three_points_to_circle(tri_edge)

            if not center:
                continue

            distance = norm(subtract(point, center))
            if point[0] > center[0] + radius:
                tri_ans.append(tri_temp.pop(j))
            elif distance < radius:
                for k in range(3):
                    p1 = tri_edge[k]
                    p2 = tri_edge[(k + 1) % 3]
                    if p1 < p2:
                        edge_buffer.append([p1, p2])
                    else:
                        edge_buffer.append([p2, p1])
                tri_temp.pop(j)

        edge_buffer.sort()
        edge_buffer = [e for i, e in enumerate(edge_buffer) if i == 0 or e != edge_buffer[i - 1]]

        for edge in edge_buffer:
            tri_temp.append([edge[0], edge[1], point])

    tri_ans.extend(tri_temp)
    tri_ans = [
        tri for tri in tri_ans
        if all(min_point[0] <= p[0] <= max_point[0] and min_point[1] <= p[1] <= max_point[1] for p in tri)
    ]

    return tri_ans


# Example Usage
points = [
    [0, 0],
    [1, 0],
    [0, 1],
    [1, 1],
    [0.5, 0.5]
]

triangles = delaunay(points)
print("Triangles:", triangles)
