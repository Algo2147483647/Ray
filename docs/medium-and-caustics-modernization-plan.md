# Medium And Caustics Modernization Plan

This document tracks medium and caustics modernization after spectral materials. The explicit medium/boundary model has been partially implemented; the remaining major work is volume attenuation/scattering and caustic-capable light transport for prism and glass-on-screen validation scenes.

Status after the controller/material refactor:

- Implemented: `medium.Registry`, constant and Cauchy IOR models, `medium.Boundary`, priority-aware `medium.Stack`, object-level `medium_boundary`, and renderer-side incident/transmitted eta resolution.
- Still planned: homogeneous volume attenuation, participating-media scattering, direct-light/specular-chain sampling, photon mapping, BDPT, or another caustic-capable integrator.

## 1. Current State

The renderer now handles refraction with a hybrid compatibility model:

```text
Ray.MediumStack + object MediumBoundary
  -> ShadingContext.EtaIncident / EtaTransmit
  -> SpecularDielectric.Sample
  -> BxDFSample.TransmitMedium / Eta
  -> Ray.MediumStack and Ray.RefractionIndex update
```

This supports nested dielectric IOR decisions and remains wavelength-aware through the IOR model. The remaining limitations are:

- Volume absorption is parsed but not applied during transport.
- Participating media are not implemented.
- Thin shells are represented, but advanced layered/coating behavior is not implemented.
- A diffuse receiver behind a prism cannot be lit efficiently by the current camera-only path tracer because the important paths are specular caustics.

The modern target should separate three concepts:

```text
Surface BSDF:
  how light scatters at a boundary

Medium:
  what light travels through between boundaries

Transport integrator:
  how paths are sampled, connected, and accumulated
```

## 2. Design Goals

1. Support nested dielectric boundaries deterministically.
2. Keep old scenes renderable with default air/glass behavior.
3. Keep IOR spectral: eta may depend on wavelength.
4. Make the ray state carry medium identity, not only a scalar eta.
5. Prepare for absorption and volumes without implementing full volume rendering first.
6. Add a caustic path that can validate prism dispersion on a diffuse screen.
7. Avoid putting wavelength sampling, medium stack mutation, or caustic sampling inside BxDFs.

## 3. Core Types

### 3.1 Medium

```go
type MediumID uint32

type Medium interface {
    ID() MediumID
    Name() string
    IOR(ctx WavelengthContext) float64
}
```

The current implementation keeps homogeneous dielectric media focused on IOR:

```go
type Homogeneous struct {
    id     MediumID
    name   string
    eta    Model
}
```

`sigma_a` and `sigma_s` are still accepted by scene JSON, but they are schema placeholders until volume transmittance is implemented.

### 3.2 Medium Stack

```go
type Stack struct {
    entries []StackEntry
}

func (s Stack) Current() MediumID
func (s *Stack) Push(id MediumID)
func (s *Stack) PushWithPriority(id MediumID, priority int)
func (s *Stack) Remove(id MediumID) bool
func (s Stack) Contains(id MediumID) bool
func (s Stack) Clone() Stack
```

The current medium is the topmost active medium. For ordinary nested closed surfaces, pushing on entry and removing on exit is enough. For overlapping surfaces, use boundary priority.

### 3.3 Medium Boundary

Objects that represent dielectric boundaries should declare what they separate:

```go
type Boundary struct {
    Outside MediumID
    Inside  MediumID
    Priority int
    Thin    bool
}
```

Rules:

- `Outside` defaults to air/vacuum.
- `Inside` defaults to the material's dielectric medium.
- `Priority` resolves overlaps; higher priority wins when multiple containers overlap.
- `Thin` means the object does not push a medium. It behaves like a window pane or coating boundary.

### 3.4 Ray State

The ray now carries a medium stack while keeping `RefractionIndex` as a scalar compatibility/cache value:

```go
type Ray struct {
    ...
    MediumStack medium.Stack
    RefractionIndex float64
    WaveLength float64
    WavelengthPDF float64
}
```

Compatibility:

- `Ray.RefractionIndex` is updated from the current medium IOR.
- Enter/exit truth comes from front-face orientation plus `medium.Boundary`.
- Scalar comparison is no longer the primary source of boundary transition truth.

## 4. Boundary Classification

At an intersection, use geometry orientation plus the object's declared boundary:

```text
front face hit:
  ray is entering object boundary

back face hit:
  ray is exiting object boundary
```

In code, the hit record should preserve the unflipped geometric normal and a `FrontFace` boolean. Rendering code may still orient the shading normal for local frames, but medium updates must use the true side of the boundary.

Recommended hit record:

```go
type SurfaceHit struct {
    Distance float64
    Point    *mat.VecDense
    GeometricNormal *mat.VecDense
    ShadingNormal   *mat.VecDense
    FrontFace bool
    Object *object.Object
}
```

Current code flips the normal early:

```go
if dot := mat.Dot(normal, ray.Direction); dot > 0 {
    normal.ScaleVec(-1, normal)
}
```

That is fine for shading, but not enough for medium decisions. The modernization should compute `FrontFace` before flipping and pass both forms forward.

## 5. Eta Selection

The dielectric BxDF should receive eta from the transport layer, not infer it by comparing `ctx.CurrentIOR` to its own inside eta.

Target context:

```go
type ShadingContext struct {
    TransportMode TransportMode
    SpectrumMode  SpectrumMode
    WavelengthNM  float64
    WavelengthsNM []float64
    WavelengthPDF float64

    EtaIncident    float64
    EtaTransmit    float64
    IncidentMedium MediumID
    TransmitMedium MediumID
    Entering       bool
}
```

Dielectric sampling becomes:

```text
FresnelDielectric(abs(cosTheta), ctx.EtaIncident, ctx.EtaTransmit)
refract using etaIncident / etaTransmit
return DeltaTransmission with TransmitMedium
```

The integrator, not the BxDF, mutates the ray's medium stack after a transmission event.

## 6. Spectral Interaction

Medium IOR and attenuation must evaluate under the active wavelength sample:

```text
rgb mode:
  use default wavelength only for scalar eta compatibility

hero_wavelength mode:
  eta(lambda) drives refraction angle and Fresnel

sampled mode:
  each wavelength subpath has its own eta(lambda)
```

For now, sampled mode can continue tracing independent wavelength subpaths. A later packet-style spectrum can evaluate several lambdas in one path, but it must handle wavelength-dependent directions carefully because dispersion splits directions.

This is why prism dispersion is naturally modeled as many wavelength subpaths, not as one packet ray with one shared direction.

## 7. JSON Schema

### 7.1 Medium Library

```json
{
  "media": {
    "air": {
      "type": "homogeneous",
      "ior": { "type": "constant", "eta": 1.000277 }
    },
    "bk7": {
      "type": "homogeneous",
      "ior": { "type": "cauchy", "a": 1.5046, "b": 0.00420, "c": 0.000012 }
    },
    "water": {
      "type": "homogeneous",
      "ior": { "type": "cauchy", "a": 1.322, "b": 0.003, "c": 0 }
    }
  }
}
```

### 7.2 Object Boundary

```json
{
  "name": "glass_prism",
  "shape": { "type": "mesh", "file": "prism.obj" },
  "material": {
    "surface": {
      "type": "specular_dielectric",
      "reflectance": { "type": "constant", "value": 1 },
      "transmittance": { "type": "constant", "value": 1 }
    }
  },
  "medium_boundary": {
    "outside": "air",
    "inside": "bk7",
    "priority": 10
  }
}
```

Compatibility rule:

- Existing `specular_dielectric.eta_inside` and `ior` fields create an implicit inside medium.
- Existing `eta_outside` creates or references the default outside medium.
- New scenes should prefer named media and `medium_boundary`.

## 8. Caustic Transport Plan

The current camera path tracer cannot efficiently render a prism projecting colored light onto a diffuse screen. The important path class is:

```text
Light -> specular transmission/reflection -> diffuse receiver -> camera
```

Camera tracing samples this path poorly because it must randomly find a small light through one or more specular events. The modern plan should add a light-side caustic pass before full BDPT.

### 8.1 Phase C1: Light Tracing Caustic Buffer

Add an optional caustic pass:

```text
sample light position and direction
trace through specular/delta events
when a path hits a diffuse surface, splat radiance to a caustic buffer
camera pass reads the caustic buffer on visible diffuse surfaces
```

Minimum data:

```go
type CausticSample struct {
    Position *mat.VecDense
    Normal   *mat.VecDense
    Radius   float64
    Spectrum optics.Spectrum
}
```

For the first implementation, restrict splatting to explicitly named receiver objects. This keeps prism validation predictable and avoids building a full global photon map immediately.

### 8.2 Phase C2: Photon Map

Generalize the caustic buffer into a spatial index:

```text
emit photons from lights
store caustic photons after at least one delta event
estimate irradiance by radius or k-nearest lookup
combine with camera-visible diffuse surfaces
```

Acceptance:

- Prism-on-screen scenes show stable colored caustic bands.
- Increasing photon count reduces noise without changing band order.
- Blue bends more than red for normal dispersion glass.

### 8.3 Phase C3: BDPT

Bidirectional path tracing can be added after the material/medium state is reliable:

```text
build camera subpath
build light subpath
connect non-delta vertices with MIS
handle delta chains with special rules
```

BDPT is more general than a caustic photon pass, but it is a larger integration change. It should not be the first caustic implementation unless the renderer is already ready for MIS, light sampling PDFs, and vertex connection bookkeeping.

## 9. Implementation Phases

### Phase M1: Boundary State Without Behavior Change

Tasks:

- Add `medium` package with `MediumID`, `Medium`, `Homogeneous`, and `Stack`.
- Add default air medium.
- Add `medium.Boundary` to objects.
- Preserve `Ray.RefractionIndex` compatibility.
- Add tests for stack push/remove/current behavior.

Acceptance:

- Existing scenes render unchanged.
- No material needs to know about a raw medium stack.

### Phase M2: Hit Record And FrontFace

Tasks:

- Add a hit record type that keeps `FrontFace`, geometric normal, and shading normal.
- Stop using flipped normals to infer medium transitions.
- Keep existing local-frame shading output the same.

Acceptance:

- Reflection/refraction tests still pass.
- Entering and exiting a closed sphere is detected deterministically.

### Phase M3: Eta From Medium Boundary

Tasks:

- Populate `EtaIncident`, `EtaTransmit`, `IncidentMedium`, `TransmitMedium`, and `Entering` in `ShadingContext`.
- Update `SpecularDielectric.Sample` to use context eta.
- Move medium-stack mutation into `TraceRay` after accepted transmission.
- Keep legacy eta fields as implicit media.

Acceptance:

- Air -> glass -> air matches current behavior.
- Air -> glass -> water -> glass -> air returns to air correctly.
- Cauchy eta still changes with wavelength.

### Phase M4: Scene Schema

Tasks:

- Parse top-level `media`.
- Parse object-level `medium_boundary`.
- Document legacy conversion rules.
- Add validation for unknown medium ids and invalid IOR.

Acceptance:

- Existing scenes keep working.
- New named media scenes can describe nested glass and water containers.

### Phase M5: Absorption Placeholder

Tasks:

- Add Beer-Lambert transmittance for homogeneous media:

```text
T = exp(-sigmaA(lambda) * distance)
```

- Apply it while the ray travels between surface hits.
- Keep scattering zero until volume phase functions are implemented.

Acceptance:

- Tinted glass thickness affects color.
- Zero absorption preserves old results.

### Phase M6: Caustic Validation Pass

Tasks:

- Add `integrator.caustics` render option.
- Emit light-side paths from area lights.
- Trace specular chains through medium boundaries.
- Splat first diffuse receiver hit to a caustic buffer.
- Composite with the camera pass for visible receiver surfaces.

Acceptance:

- The triangular prism scene can render colored bands on a white or gray screen.
- The direct-view prism scene still works without the caustic pass.
- 200x200, 1000 spp, 16-24 wavelength samples produces a non-black validation image.

## 10. Recommended Landing Order

```text
M1 medium package and stack tests
  -> M2 hit record with FrontFace
  -> M3 context eta and dielectric update
  -> M4 named media JSON schema
  -> M5 homogeneous absorption
  -> M6 light-traced caustic buffer
  -> optional photon map
  -> optional BDPT
```

The first code PR should stop after M1-M3. That gives the renderer correct nested refraction without forcing a new integrator. The caustic pass should come next because it directly validates the prism-on-screen experiment that the camera-only tracer cannot solve robustly.
