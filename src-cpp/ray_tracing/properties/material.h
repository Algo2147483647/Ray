#ifndef RAY_TRACING_MATERIAL_H
#define RAY_TRACING_MATERIAL_H

#include <vector>
#include <Eigen/Dense>
#include "../optics/geometrical_optics.h"
#include "../entity/ray.h"

using namespace Eigen;
using namespace GeometricalOptics;

namespace RayTracing {
    struct Material {        
        Vector3f color;           // Base color of the material
        bool rediate = false;
        float reflectivity = 0;             // Reflectivity coefficient [0, 1]
        float refractivity = 0;             // Refractivity coefficient [0, 1]
        float refractiveIndex = 1;          // Index of refraction for transparent materials
        float diffuseLoss = 1;
        float reflectLoss = 1;
        float refractLoss = 1;

        // Constructor
        Material() { ; }
        Material(Vector3f color) : color(color) { ; }

        bool DielectricSurfacePropagation(Ray& ray, const Vector3f& norm) {
            if (rediate) {
                ray.color = color;
                return true;
            }

            float randnum = distribution(generator);

            if (randnum <= reflectivity) {
                ray.direction = diffuseReflect(ray.direction, norm);
                ray.color *= reflectLoss;
            }
            else if (randnum <= reflectivity + refractivity) {
                ray.direction = reflect(ray.direction, norm);
                ray.color *= refractLoss;
            }
            else {
                float refractionIndex = (ray.refractivity == 1.0f) ? refractiveIndex : 1.0f / refractiveIndex;
                ray.direction = refract(ray.direction, norm, refractionIndex);
                ray.color *= 1.0f;
                ray.refractivity = refractionIndex;
            }

            // Apply base color to the ray
            ray.color = ray.color.cwiseProduct(color.cast<float>());
            return false;
        }
    };
}
#endif