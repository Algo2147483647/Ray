# Material Capability Coverage

This document describes what the current engine can represent with its implemented
scene schema and material code. It intentionally separates three things:

- material families that can be authored in JSON and rendered today,
- BxDF/BSDF primitives that exist in code,
- material models that the architecture can host but does not implement yet.

Ground-truth implementation entry points:

```text
engine/controller/factory/materials.go
engine/model/material/material.go
engine/model/material/bsdf/
engine/model/material/bxdf/
engine/model/material/emission/
engine/model/material/medium/
engine/model/material/microfacet/
engine/ray_tracing/medium_transport.go
```

## Current Material Coverage

| Material family | Current support | Recommended authoring | Notes |
| --- | --- | --- | --- |
| Matte wall, chalk, simple diffuse paint, unpolished plastic base color | Supported | `surface.type: "lambert"` | Uses the implemented Lambert diffuse BxDF. It is a good base diffuse approximation, but not an Oren-Nayar rough-diffuse model. |
| Mirror | Supported | `surface.type: "specular_reflection"` | Perfect delta reflection with spectral or RGB reflectance. |
| Ideal metal mirror | Supported | `specular_reflection` with metal-like reflectance | This is a tinted perfect mirror. It does not evaluate conductor Fresnel from eta/k. |
| Rough metal | Supported | `surface.type: "rough_conductor"` | GGX microfacet conductor with eta, k, roughness, Smith masking-shadowing, and VNDF sampling. |
| Gold, silver, copper, aluminum | Supported when eta/k are supplied | `rough_conductor` with measured or approximated eta/k | The engine does not ship a named metal database; scenes must provide eta and k. |
| Ideal glass, water, clear transparent plastic | Supported | `surface.type: "specular_dielectric"` | Perfect specular reflection/refraction selected by Fresnel. Use IOR values such as water ~1.33 or glass ~1.5. |
| Diamond or transparent crystal | Supported | `specular_dielectric` with high IOR | Constant IOR works today. Dispersion needs a `cauchy` IOR model if the material should vary by wavelength. |
| Dispersive prism glass | Supported | `specular_dielectric` plus `ior.type: "cauchy"` | The renderer samples wavelengths and the dielectric BxDF evaluates eta at the current wavelength. |
| Tinted glass | Partially supported | `specular_dielectric.transmittance` and/or medium `sigma_a` | `transmittance` applies a surface tint. Physical distance-dependent absorption should use an object `medium_boundary` with a medium that has `sigma_a`. |
| Homogeneous absorbing medium | Partially supported | `media.*.sigma_a` plus object `medium_boundary` | Beer-Lambert absorption is applied between surface hits. `sigma_s` is parsed but no participating-media scattering event is sampled yet. |
| Constant area emitter | Supported | material `emission.type: "constant"` | Objects with emission terminate the path and contribute emitted radiance. |
| Rough wall, plaster, rough ceramic | Basic approximation only | `lambert` | True rough diffuse effects such as Oren-Nayar are not implemented. |
| Plastic | Partial approximation only | `lambert` or separate glossy-only material | There is no combined diffuse plus dielectric glossy BSDF exposed by JSON. |
| Rough plastic | Not fully supported | Planned diffuse plus rough dielectric | Missing rough dielectric reflection/transmission and mixed diffuse/glossy authoring. |
| Clear-coated wood, clear-coated ceramic, car paint | Not fully supported | Planned diffuse plus clearcoat/layering | Missing clearcoat, coating/layered BSDF, and paint-flake models. |
| Frosted glass | Supported for transmission-only surfaces | `surface.type: "rough_dielectric_transmission"` | GGX rough refraction is implemented. A full rough glass BSDF should also include a rough dielectric reflection lobe. |
| Cloth and velvet | Not fully supported | Planned diffuse plus sheen/velvet | Basic diffuse can approximate body color; sheen, fiber, and retroreflection lobes are missing. |
| Skin, wax, jade, milk plastic | Not supported | Planned subsurface/BSSRDF | No subsurface scattering or diffusion/random-walk model is implemented. |
| Hair, fur, fibers | Not supported | Planned hair/fiber BSDF | No Marschner/Disney hair model is implemented. |
| Alpha cutout and pure medium boundary surface | Not supported as a BxDF | Planned `NullBxDF` or alpha masking | `medium_boundary` exists as object metadata, but the surface still needs a material. There is no null-scattering BxDF or alpha cutout path. |

## Implemented Primitives

These are implemented by the current engine. "JSON exposed" means a scene file can
instantiate it through `materials[*].surface` or `materials[*].emission`.

| Primitive | Code type | JSON exposed | Delta | Transmission | Main use |
| --- | --- | --- | --- | --- | --- |
| Lambert diffuse reflection | `bxdf.Lambert` | `surface.type: "lambert"` | No | No | Matte diffuse surfaces. |
| Perfect specular reflection | `bxdf.SpecularReflection` | `surface.type: "specular_reflection"` | Yes | No | Mirrors and idealized metal reflection. |
| Perfect dielectric reflection/refraction | `bxdf.SpecularDielectric` | `surface.type: "specular_dielectric"` | Yes | Yes | Ideal glass, water, prisms, transparent crystals. |
| GGX rough conductor reflection | `bxdf.RoughConductor` | `surface.type: "rough_conductor"` | No | No | Rough metals with eta/k. |
| GGX rough dielectric transmission | `bxdf.RoughDielectricTransmission` | `surface.type: "rough_dielectric_transmission"` | No | Yes | Frosted glass and rough transparent plastic transmission. |
| Single-BxDF BSDF container | `bsdf.Single` | Used by all current surface types | Depends on child | Depends on child | Wraps one BxDF as a material surface. |
| Weighted BxDF mixture | `bsdf.WeightedMixture` | No | Depends on children | Depends on children | Implemented in code, but not exposed by the current JSON factory. |
| Constant emission | `emission.Constant` | `emission.type: "constant"` | N/A | N/A | Area lights and emissive objects. |
| Medium boundary and absorption | `medium.Boundary`, `medium.Stack`, `sigma_a` | `objects[*].medium_boundary`, `media` | N/A | N/A | Nested IOR decisions and homogeneous Beer-Lambert absorption. |

## BxDF Roadmap Status

The table below maps common renderer BxDF names to the current codebase. Some names
are conceptual equivalents rather than exact type names.

| Priority | BxDF / material model | Current status | Notes |
| ---: | --- | --- | --- |
| 1 | `LambertianReflection` | Implemented as `bxdf.Lambert` | JSON type is `lambert`. |
| 2 | `OrenNayarReflection` | Not implemented | Needed for rough diffuse wall, ceramic, plaster, and cloth body reflection. |
| 3 | `SpecularReflection` | Implemented | JSON type is `specular_reflection`. |
| 4 | `SpecularTransmission` | Covered inside `SpecularDielectric`, not separate | The dielectric BxDF samples delta transmission and reflection with Fresnel. |
| 5 | `FresnelSpecular` | Covered inside `SpecularDielectric`, not separate | There is no separate JSON or Go BxDF named `FresnelSpecular`. |
| 6 | `MicrofacetReflection` | Partially implemented | `RoughConductor` is a GGX microfacet conductor. Generic dielectric glossy reflection is not implemented. |
| 7 | `MicrofacetTransmission` | Implemented for dielectric transmission | `RoughDielectricTransmission` covers GGX rough refraction, but not the matching reflection lobe. |
| 8 | `DisneyDiffuse` | Not implemented | Planned PBR diffuse refinement. |
| 9 | `DisneySheen` | Not implemented | Needed for cloth/fabric grazing highlights. |
| 10 | `ClearcoatReflection` | Not implemented | Needed for car paint, glazed ceramic, and varnished wood. |
| 11 | `ThinDielectric` | Not implemented | Needed for soap bubbles, thin glass, and thin-film-like approximations. |
| 12 | `SubsurfaceApproxBxDF` / BSSRDF | Not implemented | Needed for skin, wax, jade, and milk plastic. |
| 13 | `HairBxDF` | Not implemented | Needed for hair, fur, and fibers. |
| 14 | `VelvetBxDF` | Not implemented | Needed for velvet and strong retroreflection fabrics. |
| 15 | `NullBxDF` | Not implemented | Needed for alpha cutout and pure medium boundary traversal. |

## Authoring Guidance

Use `lambert` for any simple matte base surface. Use `specular_reflection` only
when the surface should be a perfect mirror. Use `rough_conductor` for metals with
finite roughness and eta/k data. Use `specular_dielectric` for ideal transparent
interfaces, and `rough_dielectric_transmission` when the transmitted ray should be
spread by a rough dielectric interface. Add object-level `medium_boundary` and a
named medium when a transparent object should participate in nested IOR tracking
or distance-dependent absorption.

When a desired real-world material requires multiple lobes, such as plastic
(`diffuse + glossy`), car paint (`diffuse + flakes + clearcoat`), or cloth
(`diffuse + sheen`), the current JSON schema cannot author that full material yet
even though the BSDF layer has a code-level `WeightedMixture` container.
