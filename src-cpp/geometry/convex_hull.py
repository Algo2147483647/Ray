import numpy as np

def convex_hull(points):
    points = np.array(points)
    n = len(points)

    if n <= 3:
        return points.tolist()

    # Find the bottommost point (and leftmost in case of ties)
    min_idx = np.argmin(points[:, 1])
    for i in range(n):
        if points[i, 1] == points[min_idx, 1] and points[i, 0] < points[min_idx, 0]:
            min_idx = i

    # Swap the minimum point to the front
    points[[0, min_idx]] = points[[min_idx, 0]]
    min_point = points[0]

    # Sort points by polar angle with respect to the bottommost point
    def polar_angle_sort(point):
        vec = point - min_point
        return (np.arctan2(vec[1], vec[0]), np.linalg.norm(vec))

    points = np.array([points[0]] + sorted(points[1:], key=polar_angle_sort))

    # Construct the convex hull using a stack
    hull = [points[0], points[1]]

    for i in range(2, n):
        hull.append(points[i])

        # Remove the last point in hull if it makes a clockwise turn
        while len(hull) > 2:
            cross = np.cross(hull[-1] - hull[-2], hull[-3] - hull[-2])
            if cross > 0:  # Left turn
                break
            hull.pop(-2)  # Remove the second-to-last point

    return hull
