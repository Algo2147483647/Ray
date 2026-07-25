# Geometry Benchmark Matrix

This document describes the visible geometry cells in the `geometry-benchmark-matrix` scene.

Formula convention: unless noted otherwise, $x,y,z$ are local coordinates after the object's `center` and `scale` transform. Algebraic and implicit surfaces are written locally as $F(x,y,z)=0$. The Formula / Definition column records the intrinsic geometric identity of the cell: shape class, degree, radii/side lengths, topology, or combinatorics, without material or renderer-specific details.

Coordinate system: cell coordinates follow the camera view. The lower-left cell in the rendered image is `(1, 1)`; column coordinates increase from left to right, and row coordinates increase from bottom to top. This reindexing is derived from the original Array by `new row = 7 - original row` and `new column = 8 - original column`; the Array origin and deltas are reversed accordingly so every geometry remains in its original world-space position.

Placeholder cells use clear-glass spheres with radius $R=0.16$. Cell r6-c1 is a grouped cell containing the original circle and triangle.

r1-c1: Barth sextic surface
r1-c2: Togliatti quintic surface
r1-c3: Clebsch diagonal cubic surface
r1-c4: Gyroid minimal-surface
r1-c5: Chmutov surface / Banchoff form
r1-c6: Interlocked double Gyroid
r1-c7: Kummer surface

r2-c1: Ding-Dong surface / bell-shaped self-intersecting cubic
r2-c2: Transparent placeholder sphere
r2-c3: Singular Quartic Surface with a Double Line
r2-c4: Roman surface
r2-c5: Tanglecube
r2-c6: Dupin cyclide
r2-c7: Tilted torus

r3-c1: Small stellated dodecahedron
r3-c2: Triangular prism
r3-c3: Regular icosahedron
r3-c4: Regular dodecahedron
r3-c5: Regular Octahedron
r3-c6: Cube
r3-c7: Regular Tetrahedron

r4-c1: Cubic cusp / Whitney-umbrella-like surface
r4-c2: Transparent placeholder sphere
r4-c3: Transparent placeholder sphere
r4-c4: Surface of revolution by parabola
r4-c5: Transparent placeholder sphere
r4-c6: Togliatti / Dervish quintic nodal surface
r4-c7: Anisotropic quartic torus / spindle surface

r5-c1: Transparent placeholder sphere
r5-c2: Transparent placeholder sphere
r5-c3: Four-lobe near-separation metaballs
r5-c4: Transparent placeholder sphere
r5-c5: Transparent placeholder sphere
r5-c6: Szilassi polyhedron
r5-c7: Transparent placeholder sphere

r6-c1: Grouped finite disk and triangle
r6-c2: Example vertical coaxial cone/paraboloid group
r6-c3: Example superellipsoid/amethyst group
r6-c4: General shifted quadric with linear terms
r6-c5: Elliptic paraboloid quadric
r6-c6: Hyperbolic paraboloid quadric
r6-c7: Hyperboloid quadric

## Row 1

| Cell  | Mathematical Geometry | Formula / Definition | Material |
| ----- | --------------------- | -------------------- | -------- |
| r1-c1 | Barth sextic surface | Barth sextic $4(\phi^2x^2-y^2)(\phi^2y^2-z^2)(\phi^2z^2-x^2)-(1+2\phi)(x^2+y^2+z^2-1)^2=0$, $\phi=(1+\sqrt5)/2$. | `porcelain-glaze-cyan-blue`: pale cyan specular reflection, reflectance `[0.62, 0.82, 0.78]`. |
| r1-c2 | Togliatti quintic surface | Togliatti quintic $F_5(x,y,z)=0$, a degree-$5$ nodal algebraic surface. | `grass-rough-metal`: grass-colored rough conductor, eta `[1.35, 0.35, 1.55]`, k `[1.45, 3.2, 1.65]`, roughness `0.42`. |
| r1-c3 | Clebsch diagonal cubic surface | Clebsch diagonal cubic $81(x^3+y^3+z^3)-189(x^2y+x^2z+xy^2+y^2z+xz^2+yz^2)+54xyz-9(x^2+y^2+z^2)+126(xy+xz+yz)-9(x+y+z)+1=0$. | `pale-pink-sheen`: pale pink specular reflection, reflectance `[0.96, 0.68, 0.76]`. |
| r1-c4 | Gyroid minimal-surface | Gyroid level set $G_{3.3}(x,y,z)=0.08$, where $G_k=\sin(kx)\cos(ky)+\sin(ky)\cos(kz)+\sin(kz)\cos(kx)$. | `porcelain-glaze-cyan-blue`: specular reflection, reflectance `[0.62, 0.82, 0.78]`. |
| r1-c5 | Chmutov surface / Banchoff form | Degree-$8$ Chmutov surface in Banchoff form $T_8(x)+T_8(y)+T_8(z)=0$, with $T_8(u)=128u^8-256u^6+160u^4-32u^2+1$. | `pale-blue-sheen-crystal`: specular dielectric crystal with transmittance `[0.62, 0.78, 0.92]`, eta `1.5`. |
| r1-c6 | Interlocked double Gyroid | Parallel gyroid pair $G_{4.4}(x,y,z)=\pm0.24$, two interleaved triply periodic level surfaces. | `pale-blue-sheen`: pale blue specular reflection, reflectance `[0.62, 0.78, 0.92]`; `pale-red-sheen`: pale red specular reflection, reflectance `[1, 0.56, 0.52]`. |
| r1-c7 | Kummer surface | Cubic-symmetric quartic $x^4+y^4+z^4-1.15(x^2y^2+x^2z^2+y^2z^2)-0.28(x^2+y^2+z^2)+0.08=0$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |

## Row 2

| Cell  | Mathematical Geometry | Formula / Definition | Material |
| ----- | --------------------- | -------------------- | -------- |
| r2-c1 | Ding-Dong surface / bell-shaped self-intersecting cubic | Ding-Dong cubic $x^2+y^2+z^3-z^2=0$, a rotationally symmetric self-intersecting surface. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r2-c2 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r2-c3 | Singular Quartic Surface with a Double Line | Quartic saddle surface $x^2-y^2z^2+0.08z^4=0$. | `marble-cool`: Lambertian cool marble tone, albedo `[0.78, 0.8, 0.76]`. |
| r2-c4 | Roman surface | Steiner Roman surface $x^2y^2+y^2z^2+z^2x^2-xyz=0$, rotated so its local $[1,1,1]$ threefold face axis points toward the camera, then rolled about the front-back world-$x$ axis so one triangular point faces upward; left unbounded. | `pale-pink-sheen`: pale pink specular reflection, reflectance `[0.96, 0.68, 0.76]`. |
| r2-c5 | Tanglecube | Quartic superquadric shell $x^4+y^4+z^4-5(x^2+y^2+z^2)+11.8=0$, moved from r2-c2. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |
| r2-c6 | Dupin cyclide | Equation-form Dupin cyclide from an inverted torus: $\left(1-2dx+C\rho^2\right)^2-4R^2\left((x-d\rho^2)^2+y^2\right)=0$, where $\rho^2=x^2+y^2+z^2$, $C=d^2+R^2-r^2$, $d=1.55$, $R=0.66$, $r=0.22$. | `jade-glass`: specular dielectric, transmittance `[0.2, 0.88, 0.55]`, eta `1.57`. |
| r2-c7 | Tilted torus | Ring torus quartic $(x^2+y^2+z^2)^2-1.22(x^2+y^2)+0.78z^2+0.1521=0$, genus $1$, pitched $40^\circ$ about the $y$ axis toward the camera. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |

## Row 3

| Cell  | Mathematical Geometry | Formula / Definition | Material |
| ----- | --------------------- | -------------------- | -------- |
| r3-c1 | Small stellated dodecahedron | Dodecahedral stellation: $12$ pentagonal pyramids, triangulated into $60$ faces. | `yellow-glass`: specular dielectric, transmittance `[1, 0.86, 0.24]`, eta `1.5`. |
| r3-c2 | Triangular prism | Closed triangular prism: $2$ triangular bases plus $3$ rectangular sides, triangulated into $8$ faces. | `ultra-high-dispersion-prism-glass`: transparent specular dielectric, Cauchy IOR `a=1.33`, `b=0.12`, `c=0.004`. |
| r3-c3 | Regular icosahedron | Regular icosahedron: convex Platonic solid with $V=12$, $E=30$, $F=20$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r3-c4 | Regular dodecahedron | Chamfered cube-like closed polyhedron, triangulated into $36$ faces. | `sandstone`: Lambertian sandstone, albedo `[0.64, 0.52, 0.34]`. |
| r3-c5 | Regular Octahedron | Regular octahedron: $\lvert x\rvert+\lvert y\rvert+\lvert z\rvert=r$, a Platonic solid with $V=6$, $E=12$, $F=8$. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r3-c6 | Cube | Cube $[-0.17,0.17]^3$, side length $0.34$. | `high-dispersion-prism`: specular dielectric, Cauchy IOR `a=1.42`, `b=0.035`, `c=0`. |
| r3-c7 | Regular Tetrahedron | Tetrahedron $\operatorname{conv}(v_1,v_2,v_3,v_4)$, simplex with $V=4$, $E=6$, $F=4$. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |

## Row 4

| Cell  | Mathematical Geometry | Formula / Definition | Material |
| ----- | --------------------- | -------------------- | -------- |
| r4-c1 | Cubic cusp / Whitney-umbrella-like surface | Cubic singular surface $x^2-y^2z-0.09y^2=0$, locally $x^2=y^2(z+0.09)$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r4-c2 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r4-c3 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r4-c4 | Surface of revolution by parabola | Quartic cone $z^4=x^2+y^2$, singular at the apex. | `white-card`: diffuse warm white paper/card, albedo `[0.82, 0.82, 0.78]`. |
| r4-c5 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r4-c6 | Togliatti / Dervish quintic nodal surface | $D_5$-symmetric Togliatti/Dervish quintic $F_5(x,y,z)=0$, a nodal degree-$5$ surface. | `pale-blue-sheen`: pale blue specular reflection, reflectance `[0.62, 0.78, 0.92]`. |
| r4-c7 | Anisotropic quartic torus / spindle surface | Anisotropic quartic spindle surface $x^4+4y^4+256z^4+4x^2y^2-368x^2z^2-736y^2z^2+8x^2+16y^2+128z^2+16=0$. | `pure-white-reflection`: pure white specular reflection, reflectance `[1, 1, 1]`. |

## Row 5

| Cell  | Mathematical Geometry | Formula / Definition | Material |
| ----- | --------------------- | -------------------- | -------- |
| r5-c1 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r5-c2 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r5-c3 | Four-lobe near-separation metaballs | Gaussian implicit blend $\sum_{i=1}^{4}w_i e^{-2.6\lVert p-p_i\rVert^2}-0.52=0$, with two lateral lobes, one upper lobe, and one offset rear lobe arranged near disconnection. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r5-c4 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r5-c5 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r5-c6 | Szilassi polyhedron | Szilassi graph on a toroidal polyhedron: genus $1$, $V=14$, $E=21$, $F=7$. | `green-glass`: specular dielectric, transmittance `[0.34, 1, 0.52]`, eta `1.5`. |
| r5-c7 | Transparent placeholder sphere | Clear-glass sphere $S_R^2$, $R=0.16$, used as a placeholder for this cell. The previous diagonal ridged torus is retained in `geo-r05-c07-diagonal-ridged-torus-backup.json`. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |

## Row 6

| Cell  | Mathematical Geometry | Formula / Definition | Material |
| ----- | --------------------- | -------------------- | -------- |
| r6-c1 | Grouped finite disk and triangle | `shape: "group"` combines a scaled disk $D_R^2$, $R=0.22$ before group scale, with a filled triangle in the same cell. | `emissive-cyan`: constant cyan emission, color `[0.5, 3.6, 4.8]`; `fabric-teal`: Lambertian teal fabric, albedo `[0.05, 0.42, 0.38]`. |
| r6-c2 | Example vertical coaxial cone/paraboloid group | `shape: "group"` retained as a matrix-cell local legacy example, combining a vertical finite cone opening reference with a coaxial vertical cylinder. | `rough-gold`: rough conductor on the cone; `ultra-high-dispersion-prism-glass`: transparent Cauchy dielectric on the cylinder. |
| r6-c3 | Example superellipsoid/amethyst group | `shape: "group"` copied from `geo_example.json` into matrix-cell local placement, combining an $L^4$ superellipsoid and a same-center, same-radius amethyst sphere. | `clear-glass`: transparent superellipsoid; `pale-blue-sheen-crystal`: specular dielectric sphere. |
| r6-c4 | General shifted quadric with linear terms | Shifted second-degree quadric $0.25x^2+0.49y^2+z^2+x+y+z+1=0$. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r6-c5 | Elliptic paraboloid quadric | Elliptic paraboloid $z=11.1111x^2+16.0000y^2-0.18$. | `ceramic-white`: Lambertian ceramic white, albedo `[0.86, 0.84, 0.78]`. |
| r6-c6 | Hyperbolic paraboloid quadric | Doubly ruled saddle $z=10.9375(x^2-y^2)$. | `brushed-aluminum`: rough conductor, eta `[1.44, 0.93, 0.62]`, k `[7.4, 6.6, 5.3]`, roughness `0.32`. |
| r6-c7 | Hyperboloid quadric | Second-degree hyperboloid $82.6446x^2+44.4444y^2-17.3611z^2-1=0$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |

This example runs the geometry benchmark matrix scene through `studio` first, then lets `engine` render the generated intermediate JSON.

Run the small example:

```cmd
run_example.cmd
```

Run the full matrix:

```cmd
run.cmd
```

Additional engine-compatible flags can be appended to either command.
