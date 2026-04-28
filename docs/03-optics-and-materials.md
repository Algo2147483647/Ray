# Optics and Materials

This document describes the physical ideas used when a ray reaches a surface. The code is primarily based on geometrical optics with a stochastic surface-interaction model.

## 1. What a Ray Represents

In this project, a ray carries more than just position and direction. It stores:

- origin,
- direction,
- RGB color weight,
- wavelength,
- refractive index of the current medium,
- a debug flag.

This means a ray is both:

- a geometric object in space,
- a compact state vector for optical transport.

Relevant code:

- `model/optics/ray.go`

## 2. Surface Interaction Model

When a ray hits a material, the material randomly chooses one of three behaviors:

1. reflection,
2. refraction,
3. diffuse scattering.

The probabilities are controlled by:

- `Reflectivity`,
- `Refractivity`,
- the remaining probability mass for diffuse response.

This is a probabilistic transport model. Each bounce follows one sampled branch rather than splitting into multiple simultaneous child rays.

Relevant code:

- `model/optics/material.go`

## 3. Reflection

The reflected direction is computed from the standard mirror law:

```text
r = d - 2 (n · d) n
```

where:

- `d` is the incident direction,
- `n` is the outward unit normal,
- `r` is the reflected unit direction.

This formula is the vector form of "angle of incidence equals angle of reflection."

Relevant code:

- `utils/geometrical_optics.go`

## 4. Refraction and Snell's Law

Refraction changes direction when a ray crosses an interface between two media with different refractive indices.

The implementation computes:

```text
eta = n1 / n2
```

where:

- `n1` is the current medium index stored on the ray,
- `n2` is the target medium index provided by the material.

The code then uses a vector refraction formula derived from Snell's law. The key scalar relation is:

```text
n1 sin(theta1) = n2 sin(theta2)
```

This is why the algorithm first computes the transmitted sine term through `eta`.

Relevant code:

- `utils/geometrical_optics.go`
- `model/optics/material.go`

## 5. Total Internal Reflection

When light goes from a higher index medium to a lower one, refraction may become impossible beyond the critical angle.

The implementation checks:

```text
sin^2(theta_t) > 1
```

If this is true, the refracted solution is not real, and the event becomes pure reflection.

This is the classic total internal reflection condition.

Relevant code:

- `utils/geometrical_optics.go`
- `model/optics/material.go`

## 6. Fresnel Effect via Schlick Approximation

Even when refraction is possible, real dielectric surfaces do not reflect a constant fraction of light. Reflectance depends on angle.

The project uses the Schlick approximation:

```text
R(theta) = R0 + (1 - R0) (1 - cos(theta))^5
```

with:

```text
R0 = ((n1 - n2) / (n1 + n2))^2
```

This is a very common physically inspired approximation because it captures the increase in reflectivity near grazing angles without requiring the full Fresnel equations.

Relevant code:

- `utils/geometrical_optics.go`

## 7. Diffuse Reflection

If the ray is neither reflected nor refracted, the surface performs diffuse scattering.

In 3D, the code samples a cosine-weighted hemisphere around the normal:

- draw two random numbers,
- convert them to polar coordinates,
- build a local tangent-bitangent-normal basis,
- map the local sample into world coordinates.

This sampling pattern is physically meaningful because Lambertian reflection is naturally weighted by the cosine of the outgoing angle with respect to the normal.

Relevant code:

- `utils/geometrical_optics.go`

## 8. A 4D Diffuse Generalization

The project also includes `DiffuseReflect4D`, which constructs a random direction in the tangent subspace orthogonal to the normal and blends it with the normal direction.

This is mathematically notable because it generalizes diffuse scattering ideas beyond ordinary 3D space. Even though the main runtime dimension is currently 3, this function shows that the project is exploring higher-dimensional ray transport concepts.

Relevant code:

- `utils/geometrical_optics.go`

## 9. Emissive Materials

If a material is marked as radiating, it behaves as a light source and terminates the ray path.

Two emission modes exist:

- default emission: multiply ray color by the material color,
- directional emission: only rays aligned closely enough with the surface normal receive energy.

The directional-light mode effectively models a highly angular emitter rather than an ideal isotropic surface source.

Relevant code:

- `model/optics/material.go`

## 10. Material Color as Spectral Weighting

After the interaction type is chosen, the ray color is multiplied component-wise by the material color or by a color function:

```text
C_out = C_in * C_material
```

This is not a full spectral BRDF, but it is an effective RGB attenuation model. It makes the material color act like a channel-wise transmittance or reflectance factor.

Relevant code:

- `model/optics/material.go`
- `model/optics/color_func_lib.go`

## 11. Wavelength and Monochromatic Conversion

The renderer can convert a ray into a monochromatic ray by sampling a wavelength in the visible range:

```text
380 nm <= lambda <= 750 nm
```

The sampled wavelength is mapped to RGB using a spectral approximation. The resulting RGB is normalized and scaled before modulating the ray color.

This is an important bridge between:

- a scalar physical quantity, wavelength,
- a display-oriented quantity, RGB color.

Relevant code:

- `model/optics/ray.go`

## 12. Dispersion with the Cauchy Formula

If a material stores a three-parameter refractive-index vector, the project interprets it as Cauchy-dispersion coefficients:

```text
n(lambda) = A + B / lambda^2 + C / lambda^4
```

This means the refractive index becomes wavelength-dependent. Rays of different wavelengths bend by different amounts, which is the core mechanism behind chromatic dispersion in transparent materials.

This is one of the strongest pieces of actual physics in the renderer because it models a real optical effect rather than a purely visual trick.

Relevant code:

- `utils/geometrical_optics.go`
- `model/optics/material.go`

## 13. Medium Tracking

Each ray stores its current refractive index. When the ray refracts into a material, that value is updated. If the material's computed index matches the ray's current index, the code interprets the event as leaving the material and returns the index to air:

```text
n = 1.0
```

This is a simple inside-outside medium model. It avoids having to explicitly store a full stack of nested media, though that also means it is best suited to relatively simple interface arrangements.

Relevant code:

- `model/optics/material.go`

## 14. Energy Attenuation

The project introduces three user-controlled loss factors:

- `DiffuseLoss`
- `ReflectLoss`
- `RefractLoss`

These scale ray color after the chosen interaction. Physically, they act like coarse energy-retention factors and allow the user to damp repeated bounces.

This is not derived from first-principles radiometry, but it is a practical control mechanism for shaping brightness and stability.

Relevant code:

- `model/optics/material.go`

## 15. What Kind of Optical Model This Is

The project is best described as:

- a **geometrical optics** renderer,
- with **probabilistic dielectric and diffuse surface interactions**,
- plus **optional wavelength-dependent refraction**.

It does **not** currently model:

- wave interference,
- diffraction,
- polarization,
- full spectral power distributions,
- exact energy-conserving BSDF integration.

That boundary is important: the project contains real optics, but specifically *ray-based* optics.

## 16. Summary

The optics subsystem embeds the following physical knowledge:

- mirror reflection,
- Snell refraction,
- total internal reflection,
- angle-dependent Fresnel reflectance,
- Lambert-style diffuse scattering,
- emissive surfaces,
- wavelength sampling,
- Cauchy dispersion,
- simple medium tracking.

Together, these ideas define the renderer's physical behavior at every bounce.
