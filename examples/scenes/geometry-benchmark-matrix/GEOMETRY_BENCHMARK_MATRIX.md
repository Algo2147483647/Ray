# Geometry Benchmark Matrix

This document describes the visible geometry cells in the `geometry-benchmark-matrix` scene.

Formula convention: unless noted otherwise, $x,y,z$ are local coordinates after the object's `center` and `scale` transform. Algebraic and implicit surfaces are written locally as $F(x,y,z)=0$. The Formula / Definition column records the intrinsic geometric identity of the cell: shape class, degree, radii/side lengths, topology, or combinatorics, without material or renderer-specific details.

## Row 1

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r1-c1 | Finite disk / circle | Closed Euclidean disk $D_R^2=\{u^2+v^2\le R^2\}$, $R=0.22$; boundary is the circle $S_R^1$. | `emissive-cyan`: constant cyan emission, color `[0.5, 3.6, 4.8]`. |
| r1-c2 | Oblique finite cylinder | Circular cylinder $S_R^1\times[-h/2,h/2]$, with radius $R=0.10$ and height $h=0.48$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r1-c3 | Quartic elliptic cylinder / quartic cone section | Quartic cone-cylinder section $1525.8789x^4+2603.0820y^4-34.6021z^2=0$. | `frosted-glass`: rough dielectric transmission, transmittance `[0.82, 0.9, 1]`, eta `1.5`, roughness `0.5`. |
| r1-c4 | Rectangular cuboid | Orthotope $ [-0.09,0.09]\times[-0.21,0.21]\times[-0.11,0.11]$, side lengths $0.18,0.42,0.22$. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |
| r1-c5 | Superellipse quartic column | Superellipse cylinder $3501.2780x^4+3501.2780y^4=1$; cross-section has $L^4$-symmetry. | `marble-cool`: Lambertian cool marble tone, albedo `[0.78, 0.8, 0.76]`. |
| r1-c6 | Quartic saddle sheet | Quartic saddle graph $z=457.7637x^4-359.1911y^4$. | `plastic-blue`: Lambertian blue, albedo `[0.05, 0.16, 0.75]`. |
| r1-c7 | Quartic elliptic paraboloid | Convex quartic paraboloid $z=579.8340x^4+989.1709y^4-0.19$. | `ceramic-white`: Lambertian ceramic white, albedo `[0.86, 0.84, 0.78]`. |

## Row 2

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r2-c1 | General shifted quadric with linear terms | Shifted second-degree quadric $0.25x^2+0.49y^2+z^2+x+y+z+1=0$. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r2-c2 | Hyperbolic paraboloid quadric | Doubly ruled saddle $z=10.9375(x^2-y^2)$. | `brushed-aluminum`: rough conductor, eta `[1.44, 0.93, 0.62]`, k `[7.4, 6.6, 5.3]`, roughness `0.32`. |
| r2-c3 | Rectangular cuboid | Orthotope $ [-0.21,0.21]\times[-0.12,0.12]\times[-0.10,0.10]$, side lengths $0.42,0.24,0.20$. | `plastic-blue`: Lambertian blue, albedo `[0.05, 0.16, 0.75]`. |
| r2-c4 | Triangle | Filled 2-simplex $\Delta^2=\operatorname{conv}(p_1,p_2,p_3)$. | `fabric-teal`: Lambertian teal fabric, albedo `[0.05, 0.42, 0.38]`. |
| r2-c5 | Elliptic quadric / clipped ellipsoid-like surface | Elliptic quadric $30.8642x^2+30.8642y^2-8.6505z^2-2.9412z-0.25=0$. | `pearl`: Lambertian pearl tone, albedo `[0.92, 0.84, 0.68]`. |
| r2-c6 | Szilassi polyhedron | Szilassi graph on a toroidal polyhedron: genus $1$, $V=14$, $E=21$, $F=7$. | `green-glass`: specular dielectric, transmittance `[0.34, 1, 0.52]`, eta `1.5`. |
| r2-c7 | Hyperboloid quadric | One-sheet hyperboloid $82.6446x^2+44.4444y^2-17.3611z^2=1$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |

## Row 3

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r3-c1 | Cubic cusp / Whitney-umbrella-like surface | Cubic singular surface $x^2-y^2z-0.09y^2=0$, locally $x^2=y^2(z+0.09)$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r3-c2 | Elliptic paraboloid quadric | Elliptic paraboloid $z=11.1111x^2+16.0000y^2-0.18$. | `ceramic-white`: Lambertian ceramic white, albedo `[0.86, 0.84, 0.78]`. |
| r3-c3 | Vertical finite cone | Right circular cone $x^2+y^2=0.36(z+1)^2$, locally truncated to finite height. | `pure-white-reflection`: pure white specular reflection, reflectance `[1, 1, 1]`. |
| r3-c4 | Roman surface / Steiner quartic | Steiner Roman surface $x^2y^2+y^2z^2+z^2x^2-xyz=0$, a self-intersecting quartic. | `pale-pink-sheen`: pale pink specular reflection, reflectance `[0.96, 0.68, 0.76]`. |
| r3-c5 | Tilted finite disk / circle | Closed Euclidean disk $D_R^2=\{u^2+v^2\le R^2\}$, $R=0.23$; boundary is $S_R^1$. | `amber-glass`: specular dielectric, transmittance `[1, 0.58, 0.18]`, eta `1.48`. |
| r3-c6 | Oblique finite cylinder | Circular cylinder $S_R^1\times[-h/2,h/2]$, with radius $R=0.12$ and height $h=0.45$. | `matte-titanium`: rough conductor, eta `[2.7, 2.2, 1.8]`, k `[3.1, 2.8, 2.5]`, roughness `0.55`. |
| r3-c7 | Anisotropic quartic torus / spindle surface | Anisotropic quartic spindle surface $x^4+4y^4+256z^4+4x^2y^2-368x^2z^2-736y^2z^2+8x^2+16y^2+128z^2+16=0$. | `pure-white-reflection`: pure white specular reflection, reflectance `[1, 1, 1]`. |

## Row 4

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r4-c1 | Small stellated dodecahedron | Dodecahedral stellation: $12$ pentagonal pyramids, triangulated into $60$ faces. | `yellow-glass`: specular dielectric, transmittance `[1, 0.86, 0.24]`, eta `1.5`. |
| r4-c2 | Triangular prism | Closed triangular prism: $2$ triangular bases plus $3$ rectangular sides, triangulated into $8$ faces. | `ultra-high-dispersion-prism-glass`: transparent specular dielectric, Cauchy IOR `a=1.33`, `b=0.12`, `c=0.004`. |
| r4-c3 | Regular icosahedron | Regular icosahedron: convex Platonic solid with $V=12$, $E=30$, $F=20$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r4-c4 | Regular dodecahedron | Chamfered cube-like closed polyhedron, triangulated into $36$ faces. | `sandstone`: Lambertian sandstone, albedo `[0.64, 0.52, 0.34]`. |
| r4-c5 | Regular Octahedron | Regular octahedron: $\lvert x\rvert+\lvert y\rvert+\lvert z\rvert=r$, a Platonic solid with $V=6$, $E=12$, $F=8$. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r4-c6 | Cube | Cube $[-0.17,0.17]^3$, side length $0.34$. | `high-dispersion-prism`: specular dielectric, Cauchy IOR `a=1.42`, `b=0.035`, `c=0`. |
| r4-c7 | Regular Tetrahedron | Tetrahedron $\operatorname{conv}(v_1,v_2,v_3,v_4)$, simplex with $V=4$, $E=6$, $F=4$. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |

## Row 5

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r5-c1 | Togliatti quintic surface | Togliatti quintic $F_5(x,y,z)=0$, a degree-$5$ nodal algebraic surface. | `grass-rough-metal`: grass-colored rough conductor, eta `[1.35, 0.35, 1.55]`, k `[1.45, 3.2, 1.65]`, roughness `0.42`. |
| r5-c2 | Tanglecube | Quartic superquadric shell $x^4+y^4+z^4-5(x^2+y^2+z^2)+11.8=0$. | `plastic-red`: Lambertian red, albedo `[0.85, 0.05, 0.035]`. |
| r5-c3 | Singular Quartic Surface with a Double Line | Quartic saddle surface $x^2-y^2z^2+0.08z^4=0$. | `marble-cool`: Lambertian cool marble tone, albedo `[0.78, 0.8, 0.76]`. |
| r5-c4 | Gyroid minimal-surface | Gyroid level set $G_{3.3}(x,y,z)=0.08$, where $G_k=\sin(kx)\cos(ky)+\sin(ky)\cos(kz)+\sin(kz)\cos(kx)$. | `porcelain-glaze-cyan-blue`: specular reflection, reflectance `[0.62, 0.82, 0.78]`. |
| r5-c5 | $L^4$-unit superellipsoid | $L^4$-unit superellipsoid $x^4+y^4+z^4=1$. | `clear-glass`: specular dielectric, transmittance `[1, 1, 1]`, eta `1.5`. |
| r5-c6 | Interlocked double Gyroid | Parallel gyroid pair $G_{4.4}(x,y,z)=\pm0.24$, two interleaved triply periodic level surfaces. | `pale-blue-sheen`: pale blue specular reflection, reflectance `[0.62, 0.78, 0.92]`; `pale-red-sheen`: pale red specular reflection, reflectance `[1, 0.56, 0.52]`. |
| r5-c7 | Surface of revolution by parabola | Quartic cone $z^4=x^2+y^2$, singular at the apex. | `white-card`: diffuse warm white paper/card, albedo `[0.82, 0.82, 0.78]`. |

## Row 6

| Cell | Mathematical Geometry | Formula / Definition | Material |
| --- | --- | --- | --- |
| r6-c1 | Barth sextic surface | Barth sextic $4(\phi^2x^2-y^2)(\phi^2y^2-z^2)(\phi^2z^2-x^2)-(1+2\phi)(x^2+y^2+z^2-1)^2=0$, $\phi=(1+\sqrt5)/2$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r6-c2 | Ding-Dong surface / bell-shaped self-intersecting cubic | Ding-Dong cubic $x^2+y^2+z^3-z^2=0$, a rotationally symmetric self-intersecting surface. | `blue-glass`: specular dielectric, transmittance `[0.55, 0.78, 1]`, eta `1.52`. |
| r6-c3 | Clebsch diagonal cubic surface | Clebsch diagonal cubic $81(x^3+y^3+z^3)-189(x^2y+x^2z+xy^2+y^2z+xz^2+yz^2)+54xyz-9(x^2+y^2+z^2)+126(xy+xz+yz)-9(x+y+z)+1=0$. | `pale-pink-sheen`: pale pink specular reflection, reflectance `[0.96, 0.68, 0.76]`. |
| r6-c4 | Torus | Ring torus quartic $(x^2+y^2+z^2)^2-1.22(x^2+y^2)+0.78z^2+0.1521=0$, genus $1$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |
| r6-c5 | Togliatti / Dervish quintic nodal surface | $D_5$-symmetric Togliatti/Dervish quintic $F_5(x,y,z)=0$, a nodal degree-$5$ surface. | `pale-blue-sheen`: pale blue specular reflection, reflectance `[0.62, 0.78, 0.92]`. |
| r6-c6 | Dupin cyclide | Equation-form Dupin cyclide from an inverted torus: $\left(1-2dx+C\rho^2\right)^2-4R^2\left((x-d\rho^2)^2+y^2\right)=0$, where $\rho^2=x^2+y^2+z^2$, $C=d^2+R^2-r^2$, $d=1.55$, $R=0.66$, $r=0.22$. Stored as a shifted sparse quartic polynomial at cell center `[0.6, -1.12, 0.58]`, doubled in scale and left unbounded because it is closed. | `jade-glass`: specular dielectric, transmittance `[0.2, 0.88, 0.55]`, eta `1.57`. |
| r6-c7 | Kummer surface | Cubic-symmetric quartic $x^4+y^4+z^4-1.15(x^2y^2+x^2z^2+y^2z^2)-0.28(x^2+y^2+z^2)+0.08=0$. | `rough-gold`: rough conductor, eta `[0.17, 0.35, 1.5]`, k `[3.1, 2.7, 1.9]`, roughness `0.18`. |

