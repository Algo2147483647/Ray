from struct import *

def normalize(vector: Point) -> Point:
    """Normalizes a 3D vector."""
    return vector / np.linalg.norm(vector)


def tetrahedron(p1: Point, p2: Point, p3: Point, p4: Point):
    """Creates a tetrahedron using triangles."""
    triangle(p1, p2, p3)
    triangle(p2, p3, p4)
    triangle(p3, p4, p1)
    triangle(p4, p1, p2)


def cuboid(p_min: Point, p_max: Point):
    """Creates a cuboid using two diagonal points."""
    p_min_tmp = [p_min.copy() for _ in range(3)]
    p_max_tmp = [p_max.copy() for _ in range(3)]

    for i in range(3):
        p_min_tmp[i][i] = p_max[i]
        p_max_tmp[i][i] = p_min[i]

    for i in range(3):
        quadrangle(p_min, p_min_tmp[i], p_max_tmp[(i + 2) % 3], p_min_tmp[(i + 1) % 3])
        quadrangle(p_max, p_max_tmp[i], p_min_tmp[(i + 2) % 3], p_max_tmp[(i + 1) % 3])


def cuboid_with_center(center: Point, X: float, Y: float, Z: float):
    """Creates a cuboid centered at a given point with specified dimensions."""
    delta = np.array([X / 2, Y / 2, Z / 2])
    p_max = center + delta
    p_min = center - delta
    cuboid(p_min, p_max)


def cuboid_with_direction(center: Point, direction: Point, L: float, W: float, H: float):
    """Creates a cuboid with direction and dimensions."""
    direction = normalize(direction)
    st = center - direction * (L / 2.0)
    ed = center + direction * (L / 2.0)
    face = [
        np.array([-W / 2, -H / 2]),
        np.array([W / 2, -H / 2]),
        np.array([W / 2, H / 2]),
        np.array([-W / 2, H / 2]),
        np.array([-W / 2, -H / 2]),
    ]
    translator(st, ed, face)


def frustum(st: Point, ed: Point, r_st: float, r_ed: float, point_num: int):
    """Creates a frustum (truncated cone)."""
    pass  # Implementation can be added


def sphere(center: Point, r: float, point_num: int):
    """Creates a sphere using the marching cubes algorithm."""
    margin = r / point_num * 3
    st = np.array([-r - margin, -r - margin, -r - margin])
    ed = np.array([r + margin, r + margin, r + margin])
    n = [point_num] * 3

    triangle_set = []
    marching_cubes(
        lambda x, y, z: r**2 - (x**2 + y**2 + z**2),
        st, ed, n, triangle_set
    )
    add_triangle_set(center, triangle_set)


def sphere_segmented(center: Point, r: float, theta_num: int, phi_num: int,
                     theta_st: float = 0, theta_ed: float = 2 * np.pi,
                     phi_st: float = -np.pi / 2, phi_ed: float = np.pi / 2):
    """Creates a segmented sphere."""
    d_theta = (theta_ed - theta_st) / theta_num
    d_phi = (phi_ed - phi_st) / phi_num

    for i in range(1, theta_num + 1):
        theta = theta_st + i * d_theta

        for j in range(1, phi_num + 1):
            phi = phi_st + j * d_phi

            point = [
                r * np.cos(phi) * np.cos(theta) + center[0],
                r * np.cos(phi) * np.sin(theta) + center[1],
                r * np.sin(phi) + center[2]
            ]
            point_u = [
                r * np.cos(phi - d_phi) * np.cos(theta) + center[0],
                r * np.cos(phi - d_phi) * np.sin(theta) + center[1],
                r * np.sin(phi - d_phi) + center[2]
            ]
            point_l = [
                r * np.cos(phi) * np.cos(theta - d_theta) + center[0],
                r * np.cos(phi) * np.sin(theta - d_theta) + center[1],
                r * np.sin(phi) + center[2]
            ]
            point_ul = [
                r * np.cos(phi - d_phi) * np.cos(theta - d_theta) + center[0],
                r * np.cos(phi - d_phi) * np.sin(theta - d_theta) + center[1],
                r * np.sin(phi - d_phi) + center[2]
            ]

            triangle(np.array(point), np.array(point_u), np.array(point_l))
            triangle(np.array(point_l), np.array(point_u), np.array(point_ul))