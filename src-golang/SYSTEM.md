# Ray Tracing System Documentation

## 1. System Overview

This system is a ray tracing simulation and rendering system developed in Go language, specifically designed for optical simulation and visualization. The system can simulate light propagation, reflection, refraction and other phenomena in different materials, generating high-quality optical scene rendering images.

### 1.1 System Objectives

- Provide accurate light propagation simulation
- Support various geometric shapes and material properties
- Implement visualization of optical phenomena
- Provide research tools for optical engineers, researchers, and educators

### 1.2 Core Functions

- Ray tracing computation
- Optical object modeling (spheres, planes, cuboids, etc.)
- Scene construction and rendering
- Material and refractive index definition
- Output images or data files

## 2. System Architecture

The system adopts a layered architecture and modular design, primarily consisting of the following modules:

```
src-golang/
├── controller/          # Control layer, handling input and script parsing
├── math_lib/            # Mathematical computation library (vectors, equations, colors, etc.)
├── model/               # Optical models (shapes, materials, scenes, etc.)
├── ray_tracing/         # Core ray tracing logic
├── utils/               # Utility functions and object pools
├── ui/                  # User interface and rendering scripts
├── main.go              # System entry point
└── handler.go           # Main processing logic
```

### 2.1 Module Descriptions

#### controller (Control Layer)
Responsible for parsing JSON format scene scripts, including material and object definitions.

#### math_lib (Math Library)
Provides mathematical functions such as vector operations, geometric optics calculations, and color processing.

#### model (Model Layer)
Defines core concepts of the optical system, including:
- Shape: Definition of various geometric shapes and ray intersection calculations
- Material: Material properties and handling of light-material interactions
- Object: Combination of shapes and materials
- Scene: Complete scene containing all objects and cameras
- Camera: Defines viewing perspective and ray generation

#### ray_tracing (Ray Tracing)
Implements ray tracing algorithms, including ray generation, propagation, and pixel color calculation.

#### utils (Utilities)
Provides general utility functions, object pools, and file processing capabilities.

#### ui (User Interface)
Contains HTML interface and Python scripts for visualizing rendering results.

## 3. Core Components Detailed

### 3.1 Ray

Ray is the most fundamental concept in the system, containing the following attributes:
- Origin: Starting coordinates
- Direction: Propagation direction
- Color: Color value
- WaveLength: Wavelength (for dispersion calculations)
- RefractionIndex: Refractive index of the current medium

### 3.2 Material

Material defines the optical properties of object surfaces:
- Color: Base color
- Reflectivity: Reflection coefficient
- Refractivity: Refraction coefficient
- RefractiveIndex: Refractive index (supports dispersion)
- DiffuseLoss/ReflectLoss/RefractLoss: Light energy loss from various interactions

### 3.3 Shape

The system supports various geometric shapes:
- Sphere
- Plane
- Triangle
- Cuboid

Each shape implements the following interface methods:
- Intersect: Calculate intersection with ray
- GetNormalVector: Get normal vector at intersection point
- BuildBoundingBox: Construct bounding box

### 3.4 Camera

Camera defines the viewing perspective of the scene:
- Position: Camera position
- Direction: Viewing direction
- Up: Up vector
- FieldOfView: Field of view angle
- Width/Height: Image resolution

### 3.5 Scene

Scene contains all objects and cameras, and is the basic unit for rendering.

## 4. Workflow

### 4.1 System Startup

1. Load scene script (JSON format) from command line arguments or default path
2. Parse the script and build materials and objects
3. Create camera and set viewing parameters
4. Execute ray tracing rendering
5. Save rendering results as image files

### 4.2 Rendering Process

1. Iterate through each pixel of the image
2. Generate rays passing through that pixel from the camera
3. Trace the propagation path of rays in the scene
4. Calculate interactions between rays and objects (reflection, refraction, diffuse reflection, etc.)
5. Accumulate ray color values
6. Write final color values to the image

### 4.3 Ray Tracing Algorithm

The system employs Monte Carlo ray tracing algorithm:
1. Generate multiple random rays for each pixel for anti-aliasing
2. Trace the complete propagation path of each ray
3. Determine ray interaction methods based on material properties
4. Recursively trace reflected and refracted rays
5. Accumulate color contributions from all rays

## 5. Scene Script Format

The system uses JSON format to define scenes, containing materials and objects:

```json
{
  "materials": [
    {
      "id": "material_id",
      "color": [r, g, b],
      "reflectivity": 0.0-1.0,
      "refractivity": 0.0-1.0,
      "refractive_index": [n]
    }
  ],
  "objects": [
    {
      "id": "object_id",
      "shape": "sphere|plane|triangle|cuboid",
      "material_id": "material_id",
      "...": "shape-specific parameters"
    }
  ]
}
```

## 6. Technical Features

### 6.1 Performance Optimization

- Use object pools to reduce memory allocation
- Concurrent rendering to improve computational efficiency
- Vectorized computations to accelerate mathematical operations

### 6.2 Physical Accuracy

- Light propagation simulation based on real physical laws
- Support for dispersion effects
- Energy-conserving light interaction calculations

### 6.3 Extensibility

- Modular design facilitates adding new shapes and materials
- Plugin shading functions support complex material effects
- Configurable rendering parameters

## 7. Usage Instructions

### 7.1 Environment Requirements

- Go 1.23.0
- Python 3.x (for visualization)

### 7.2 Building and Running

```bash
# Build
go build -o src-golang main.go

# Run
go run main.go [scene_script.json]

# Or
./src-golang [scene_script.json]
```

### 7.3 Output Files

- output.png: Rendered result image
- debug_traces.json: Debug information (when debug mode is enabled)
- img.bin: Intermediate result data (optional)

## 8. Visualization Interface

The system provides a WebGL-based 3D visualization interface that allows:
- Real-time viewing of rendering results
- Adjustment of viewing angles
- Analysis of light paths

## 9. Development Guide

### 9.1 Adding New Shapes

1. Create a new shape file in the [model/shape](file:///C:/Algo/Projects/Ray/src-golang/model/shape) directory
2. Implement the [Shape](file:///C:/Algo/Projects/Ray/src-golang/model/shape/shape.go#L9-L13) interface
3. Add parsing logic in [controller/parse_shape.go](file:///C:/Algo/Projects/Ray/src-golang/controller/parse_shape.go)

### 9.2 Adding New Material Properties

1. Add new fields to the [Material](file:///C:/Algo/Projects/Ray/src-golang/model/optics/material.go#L12-L22) struct
2. Implement processing logic in the [DielectricSurfacePropagation](file:///C:/Algo/Projects/Ray/src-golang/model/optics/material.go#L33-L66) method
3. Update the JSON script format

### 9.3 Performance Optimization

- Use object pools in [utils/pools.go](file:///C:/Algo/Projects/Ray/src-golang/utils/pools.go) to reduce memory allocation
- Optimize mathematical calculations with vectorized operations
- Use concurrency appropriately to improve rendering speed