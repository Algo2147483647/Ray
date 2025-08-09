import numpy as np

def cross_product(a, b, c):
    return (b[0] - a[0]) * (c[1] - b[1]) - (b[1] - a[1]) * (c[0] - b[0])

def is_ear(a, b, c, polygon):
    # If not a convex vertex
    if cross_product(a, b, c) < 0:
        return False

    for p in polygon:
        if np.array_equal(p, a) or np.array_equal(p, b) or np.array_equal(p, c):
            continue
        # If inside the triangle
        if (cross_product(a, b, p) > 0 and
            cross_product(b, c, p) > 0 and
            cross_product(c, a, p) > 0):
            return False

    return True

def ear_clipping_triangulation(polygon):
    polygon = [list(p) for p in polygon]
    triangle_set = []

    while len(polygon) >= 3:
        n = len(polygon)
        for i in range(n):
            a = (i + n - 1) % n
            c = (i + 1) % n
            if len(polygon) == 3 or is_ear(polygon[a], polygon[i], polygon[c], polygon):
                triangle_set.append([
                    polygon[a] + [0],
                    polygon[i] + [0],
                    polygon[c] + [0]
                ])
                del polygon[i]
                break  # Start from the beginning again

    return triangle_set
