#ifndef RAY_TRACING_RAY_H
#define RAY_TRACING_RAY_H

#include <Eigen/Dense>

using namespace Eigen;

namespace RayTracing{
	struct Ray {
		Vector3f origin;
		Vector3f direction;
		Vector3f color; // Color of the ray, which will be updated
	
		Ray(Vector3f& origin, Vector3f& direct) : origin(origin), direct(direct), refractivity(1.0) {
			color = Vector3f::Ones();
		}
	};
}

#endif