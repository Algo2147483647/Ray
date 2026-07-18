# Geometry Benchmark Matrix

This document describes the visible geometry cells in the `geometry-benchmark-matrix` scene.

Formula convention: unless noted otherwise, `x`, `y`, and `z` are local coordinates after the object's `center` and `scale` transform. Algebraic and implicit surfaces are written as `F(x, y, z) = 0` and are clipped by their JSON bounds when bounds are present.

## Row 01

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r01-c01 | Finite disk / circle | Plane disk: `n . (p - c) = 0`, `norm(p - c) <= 0.22`, with normal `n = normalize(-1, 0, 0.2)`. | `emissive-cyan`: constant cyan emission, color `[0.5, 3.6, 4.8]`. |
| r01-c02 | Oblique finite cylinder | Cylinder: `norm((p - c) - ((p - c) . a)a) = 0.10`, `abs((p - c) . a) <= 0.24`, with `a = normalize(0.7, -0.25, 1)`. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r01-c03 | Quartic elliptic cylinder / quartic cone section | `1525.8789 x^4 + 2603.0820 y^4 - 34.6021 z^2 = 0`. | `frosted-glass`: rough dielectric transmission, transmittance `[0.82, 0.9, 1]`, eta `1.5`, roughness `0.5`. |
| r01-c04 | Rectangular cuboid | Box: `abs(x) <= 0.09`, `abs(y) <= 0.21`, `abs(z) <= 0.11`. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |
| r01-c05 | Superellipse quartic column | `3501.2780 x^4 + 3501.2780 y^4 - 1 = 0`. | `marble-cool`: Lambertian cool marble tone, albedo `[0.78, 0.8, 0.76]`. |
| r01-c06 | Quartic saddle sheet | `1525.8789 x^4 - 1197.3037 y^4 - 3.3333 z = 0`. | `plastic-blue`: Lambertian blue, albedo `[0.05, 0.16, 0.75]`. |
| r01-c07 | Quartic elliptic paraboloid | `1525.8789 x^4 + 2603.0820 y^4 - 2.6316 z - 0.5 = 0`. | `ceramic-white`: Lambertian ceramic white, albedo `[0.86, 0.84, 0.78]`. |

## Row 02

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r02-c01 | General shifted quadric with linear terms | `0.25 x^2 + 0.49 y^2 + z^2 + x + y + z + 1 = 0`. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r02-c02 | Hyperbolic paraboloid quadric | `39.0625 x^2 - 39.0625 y^2 - 3.5714 z = 0`. | `brushed-aluminum`: rough conductor, eta `[1.44, 0.93, 0.62]`, k `[7.4, 6.6, 5.3]`, roughness `0.32`. |
| r02-c03 | Rectangular cuboid | Box: `abs(x) <= 0.21`, `abs(y) <= 0.12`, `abs(z) <= 0.10`. | `plastic-blue`: Lambertian blue, albedo `[0.05, 0.16, 0.75]`. |
| r02-c04 | Triangle | Convex hull of three vertices: `conv(p1, p2, p3)`. | `fabric-teal`: Lambertian teal fabric, albedo `[0.05, 0.42, 0.38]`. |
| r02-c05 | Elliptic quadric / clipped ellipsoid-like surface | `30.8642 x^2 + 30.8642 y^2 - 8.6505 z^2 - 2.9412 z - 0.25 = 0`. | `pearl`: Lambertian pearl tone, albedo `[0.92, 0.84, 0.68]`. |
| r02-c06 | Horizontal finite cylinder | Cylinder: `norm((p - c) - ((p - c) . a)a) = 0.13`, `abs((p - c) . a) <= 0.22`, with `a = (1, 0, 0)`. | `amber-glass`: specular dielectric, transmittance `[1, 0.58, 0.18]`, eta `1.48`. |
| r02-c07 | Hyperboloid quadric | `82.6446 x^2 + 44.4444 y^2 - 17.3611 z^2 - 1 = 0`. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |

## Row 03

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r03-c01 | Cubic cusp / Whitney-umbrella-like surface | `x^2 - y^2 z - 0.09 y^2 = 0`. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r03-c02 | Elliptic paraboloid quadric | `30.8642 x^2 + 44.4444 y^2 - 2.7778 z - 0.5 = 0`. | `ceramic-white`: Lambertian ceramic white, albedo `[0.86, 0.84, 0.78]`. |
| r03-c03 | Vertical finite cone | Cone: `x^2 + y^2 - 0.36(z + 1)^2 = 0`, clipped to `-1 <= z <= 1` in local coordinates. | `pure-white-reflection`: pure white specular reflection, reflectance `[1, 1, 1]`. |
| r03-c04 | Roman surface / Steiner quartic | Algebraic surface `x^2y^2 + y^2z^2 + z^2x^2 - xyz = 0`, clipped to the central compact self-intersecting form. | `pale-pink-sheen`: pale pink specular reflection, reflectance `[0.96, 0.68, 0.76]`. |
| r03-c05 | Tilted finite disk / circle | Plane disk: `n . (p - c) = 0`, `norm(p - c) <= 0.23`, with `n = normalize(-0.75, 0.25, 0.6)`. | `amber-glass`: specular dielectric, transmittance `[1, 0.58, 0.18]`, eta `1.48`. |
| r03-c06 | Oblique finite cylinder | Cylinder: `norm((p - c) - ((p - c) . a)a) = 0.12`, `abs((p - c) . a) <= 0.225`, with `a = normalize(0.45, 0.65, 1)`. | `matte-titanium`: rough conductor, eta `[2.7, 2.2, 1.8]`, k `[3.1, 2.8, 2.5]`, roughness `0.55`. |
| r03-c07 | Y-axis finite cylinder | Cylinder: `x^2 + z^2 = 0.13^2`, `abs(y) <= 0.21`. | `brushed-aluminum`: rough conductor, eta `[1.44, 0.93, 0.62]`, k `[7.4, 6.6, 5.3]`, roughness `0.32`. |

## Row 04

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r04-c01 | Small stellated dodecahedron-inspired triangle mesh | Dodecahedral stellation made from 12 pentagonal pyramids, represented by 60 triangular faces. | `yellow-glass`: specular dielectric, transmittance `[1, 0.86, 0.24]`, eta `1.5`. |
| r04-c02 | Szilassi polyhedron wireframe | Toroidal polyhedron graph with 14 vertices and 21 edges, represented as glass vertex spheres plus thick finite-cylinder edges. | `green-glass`: specular dielectric, transmittance `[0.34, 1, 0.52]`, eta `1.5`. |
| r04-c03 | Regular icosahedron triangle mesh | Convex polyhedron: `conv(V)`, with 12 golden-ratio icosahedral vertices and 20 triangular faces. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r04-c04 | Beveled cube / chamfered polyhedron mesh | Closed triangle mesh: union of 36 triangular faces from the JSON vertex set. | `sandstone`: Lambertian sandstone, albedo `[0.64, 0.52, 0.34]`. |
| r04-c05 | Octahedron triangle mesh | Octahedron: `abs(x) + abs(y) + abs(z) = r`, represented by 8 triangular faces. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r04-c06 | Cube / dispersive prism | Box: `abs(x) <= 0.17`, `abs(y) <= 0.17`, `abs(z) <= 0.17`. | `high-dispersion-prism`: specular dielectric, Cauchy IOR `a=1.42`, `b=0.035`, `c=0`. |
| r04-c07 | Tetrahedron triangle mesh | Tetrahedron: `conv(v1, v2, v3, v4)`, represented by 4 triangular faces. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |

## Row 05

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r05-c01 | Togliatti / Dervish quintic nodal surface | D5-symmetric Nordstrand form `a h1 h2 h3 h4 h5 + (1 - cz)(x^2 + y^2 - 1 + rz^2)^2 = 0`, expanded as the sparse polynomial stored in `geo-r05.json`. | `pale-blue-sheen`: pale blue specular reflection, reflectance `[0.62, 0.78, 0.92]`. |
| r05-c02 | Quartic superquadric shell | `x^4 + y^4 + z^4 - 5x^2 - 5y^2 - 5z^2 + 11.8 = 0`. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |
| r05-c03 | Quartic saddle / ruled-sheet surface | `x^2 - y^2 z^2 + 0.08 z^4 = 0`. | `marble-cool`: Lambertian cool marble tone, albedo `[0.78, 0.8, 0.76]`. |
| r05-c04 | Gyroid minimal-surface approximation | `sin(3.3x) cos(3.3y) + sin(3.3y) cos(3.3z) + sin(3.3z) cos(3.3x) - 0.08 = 0`. | `porcelain-glaze-cyan-blue`: specular reflection, reflectance `[0.62, 0.82, 0.78]`. |
| r05-c05 | Quartic superellipsoid | `x^4 + y^4 + z^4 - 1 = 0`. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r05-c06 | Interlocked double gyroid | Blue gyroid: `G_4.4(x,y,z) - 0.24 = 0`; red gyroid: `G_4.4(x,y,z) + 0.24 = 0`, where `G_k = sin(kx) cos(ky) + sin(ky) cos(kz) + sin(kz) cos(kx)`. | `pale-blue-sheen`: pale blue specular reflection, reflectance `[0.62, 0.78, 0.92]`; `pale-red-sheen`: pale red specular reflection, reflectance `[1, 0.56, 0.52]`. |
| r05-c07 | Quartic cone-like surface | `z^4 - x^2 - y^2 = 0`. | `white-card`: diffuse warm white paper/card, albedo `[0.82, 0.82, 0.78]`. |

## Row 06

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r06-c01 | Barth sextic surface | Classic form: `4(phi^2 x^2 - y^2)(phi^2 y^2 - z^2)(phi^2 z^2 - x^2) - (1 + 2phi)(x^2 + y^2 + z^2 - 1)^2 = 0`, where `phi = (1 + sqrt(5)) / 2`. The JSON stores this as a sparse degree-6 polynomial. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r06-c02 | Ding-Dong surface / bell-shaped self-intersecting cubic | `x^2 + y^2 + z^3 - z^2 = 0`, clipped to expose more of the lower `z < 0` flared branch. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r06-c03 | Clebsch diagonal cubic surface | `81(x^3 + y^3 + z^3) - 189(x^2y + x^2z + xy^2 + y^2z + xz^2 + yz^2) + 54xyz - 9(x^2 + y^2 + z^2) + 126(xy + xz + yz) - 9(x + y + z) + 1 = 0`. | `pale-pink-sheen`: pale pink specular reflection, reflectance `[0.96, 0.68, 0.76]`. |
| r06-c04 | Torus quartic | `(x^2 + y^2 + z^2)^2 - 1.22(x^2 + y^2) + 0.78z^2 + 0.1521 = 0`. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r06-c05 | Anisotropic quartic torus / spindle surface | `x^4 + 4y^4 + 256z^4 + 4x^2y^2 - 368x^2z^2 - 736y^2z^2 + 8x^2 + 16y^2 + 128z^2 + 16 = 0`. | `pure-white-reflection`: pure white specular reflection, reflectance `[1, 1, 1]`. |
| r06-c06 | Dupin cyclide / inverted torus mesh | Inversion of an offset torus, sampled as a 3584-triangle Dupin cyclide mesh so one side reads thick and the opposite side pinches thin. Source torus parameters: offset `1.55`, major radius `0.66`, minor radius `0.22`. | `jade-glass`: specular dielectric, transmittance `[0.2, 0.88, 0.55]`, eta `1.57`. |
| r06-c07 | Cubic-symmetric quartic surface | `x^4 + y^4 + z^4 - 1.15(x^2y^2 + x^2z^2 + y^2z^2) - 0.28(x^2 + y^2 + z^2) + 0.08 = 0`. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |

## Standalone Example

| File | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| `geo_example.json` | Enlarged r05-c06 pale red/blue reflection double gyroid | Interlocked double gyroid test: `G_4.4(x,y,z) - 0.24 = 0` and `G_4.4(x,y,z) + 0.24 = 0`. | `pale-blue-sheen`: pale blue specular reflection, reflectance `[0.62, 0.78, 0.92]`; `pale-red-sheen`: pale red specular reflection, reflectance `[1, 0.56, 0.52]`. |
