import numpy as np

def normalize(vector):
    norm = np.linalg.norm(vector)
    return vector / norm if norm > 0 else vector

def cross(v1, v2):
    return np.cross(v1, v2)

def rotation_matrix(axis):
    axis = normalize(axis)
    x, y, z = axis
    e = np.sqrt(1 - z**2)

    if e < 1e-4:
        return np.eye(3)

    sign = -1 if x * y > 0 else 1
    a = sign * abs(y / e)
    b = -x * z / e
    c = abs(x / e)
    d = -y * z / e

    return np.array([
        [a, b, x],
        [c, d, y],
        [0, e, z]
    ])

def rotate_point(point, matrix, center):
    return center + np.dot(matrix, point - center)

def translator(start, end, points, is_closed=False):
    direction = normalize(end - start)
    rotate_mat = rotation_matrix(direction)

    for i, point in enumerate(points):
        transformed_point = np.dot(rotate_mat, np.array([point[0], point[1], 0]))
        start_point = start + transformed_point
        end_point = end + transformed_point

        if i > 0:
            print("Define Quadrangle here with:", start_point, end_point)

    if is_closed:
        print("Handle closed translation")

def rotator(center, axis, points, point_num, start_angle, end_angle, is_closed=False):
    axis = normalize(axis)
    rotate_base = rotation_matrix(axis)

    d_angle = (end_angle - start_angle) / point_num

    for i in range(point_num + 1):
        angle = start_angle + d_angle * i
        delta = np.array([np.cos(angle), np.sin(angle), 0])
        direction = np.dot(rotate_base, delta)
        direction2 = normalize(cross(direction, axis))

        rotate_mat = np.column_stack([direction, axis, direction2])

        for j in range(1, len(points)):
            p1, p2 = np.array(points[j - 1]), np.array(points[j])
            p1_rotated = rotate_point(p1, rotate_mat, center)
            p2_rotated = rotate_point(p2, rotate_mat, center)

            print("Define Triangle here with:", p1_rotated, p2_rotated)

        if is_closed:
            print("Handle closed rotation")
