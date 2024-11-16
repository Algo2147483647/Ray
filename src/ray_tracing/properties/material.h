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
        // Material properties
        Eigen::Vector3d color;           // Base color of the material
        double reflectivity;             // Reflectivity coefficient [0, 1]
        double transparency;             // Transparency coefficient [0, 1]
        double refractiveIndex;          // Index of refraction for transparent materials
        double diffuseCoefficient;       // Diffuse reflection coefficient
        double specularCoefficient;      // Specular reflection coefficient
        double shininess;                // Shininess factor for specular highlights

        // Constructor
        Material(
            const Eigen::Vector3d& col = Eigen::Vector3d(1.0, 1.0, 1.0),
            double refl = 0.0,
            double transp = 0.0,
            double refractIdx = 1.0,
            double diffCoeff = 1.0,
            double specCoeff = 0.5,
            double shiny = 32.0)
            : color(col), reflectivity(refl), transparency(transp), refractiveIndex(refractIdx),
            diffuseCoefficient(diffCoeff), specularCoefficient(specCoeff), shininess(shiny) {}
    };

    void DielectricSurfacePropagation(Material& material, Ray& ray, const Vector3f& norm) {
        float randnum = distribution(generator);

        float diffuseProbability = 1.0f - material.reflectivity - material.transparency;
        float reflectProbability = material.reflectivity;
        float refractProbability = material.transparency;

        if (randnum < diffuseProbability) {
            // Diffuse reflection
            ray.direction = diffuseReflect(ray.direction, norm);
            ray.color *= material.diffuseCoefficient;
        }
        else if (randnum < diffuseProbability + reflectProbability) {
            // Specular reflection
            ray.direction = reflect(ray.direction, norm);
            ray.color *= material.specularCoefficient;
        }
        else {
            // Refraction
            float refractionIndex = (ray.refractivity == 1.0f) ? material.refractiveIndex : 1.0f / material.refractiveIndex;
            ray.direction = refract(ray.direction, norm, refractionIndex);
            ray.color *= 1.0f; // Adjust the factor if needed based on your material's refractive properties
            ray.refractivity = refractionIndex;
        }

        // Apply base color to the ray
        ray.color = ray.color.cwiseProduct(material.color.cast<float>());
    }
}
#endif