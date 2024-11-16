#ifndef RAY_TRACING_MATERIAL_H
#define RAY_TRACING_MATERIAL_H

#include <vector>
#include <Eigen/Dense>
#include "GeometricalOptics.h"
#include "Ray.h"

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
		float randnum = GeometricalOptics::dis(gen);

		if (0 && randnum < diffuseReflectProbability) {
			ray.direct = diffuseReflect(ray.direct, norm);
			ray.color *= diffuseReflectLoss;
		}
		else if (1 || randnum < reflectProbability + diffuseReflectProbability) {
			ray.direct = reflect(ray.direct, norm);
			ray.color *= reflectLoss;
		}
		else {
			ray.refractivity = (ray.refractivity == 1.0) ? refractivity[0] : 1.0 / refractivity[0];
			ray.direct = refract(ray.direct, norm, ray.refractivity);
			ray.color *= refractLoss;
		}

		ray.color = ray.color.cwiseProduct(baseColor);
	}
}
#endif