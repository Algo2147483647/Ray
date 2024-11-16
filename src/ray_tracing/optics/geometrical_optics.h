#ifndef GEOMETERICAL_OPTICS_H
#define GEOMETERICAL_OPTICS_H


#include <Eigen/Dense>
#include <random>
#include <corecrt_math_defines.h>
#include "../consts.h"

using namespace Eigen;
using namespace std;

namespace GeometricalOptics {
	thread_local std::mt19937 generator(std::random_device{}()); 
	thread_local std::uniform_real_distribution<> distribution(0, 1);

	inline Vector3f reflect(const Vector3f& incidentRay, const Vector3f& normal) {
		return (incidentRay - 2 * normal.dot(incidentRay) * normal).normalized();
	}

	inline Vector3f refract(const Vector3f& incidentRay, const Vector3f& normal, float refractionIndexRatio) {
		// refractionIndexRatio = refractive index of the incident medium / refractive index of the outgoing medium
		float cosIncident = normal.dot(incidentRay);
		float cosTransmitted = 1 - refractionIndexRatio * refractionIndexRatio * (1 - cosIncident * cosIncident);

		if (cosTransmitted < 0) {  // Total internal reflection
			return reflect(incidentRay, normal);
		}

		return (refractionIndexRatio * incidentRay + (refractionIndexRatio * cosIncident - sqrt(cosTransmitted)) * normal).normalized();
	}

	inline Vector3f diffuseReflect(const Vector3f& incidentRay, const Vector3f& normal) {
		float randomAngle = 2 * M_PI * distribution(generator);
		float randomRadius = distribution(generator);

		Vector3f tangent = (fabs(normal[0]) > EPS) ? Vector3f::UnitY() : Vector3f::UnitX();
		Vector3f u = (cos(randomAngle) * sqrt(randomRadius) * (tangent.cross(normal)).normalized()).eval();
		Vector3f v = (sin(randomAngle) * sqrt(randomRadius) * (normal.cross(u)).normalized()).eval();

		return (sqrt(1 - randomRadius) * normal + u + v).normalized();
	}
}

#endif