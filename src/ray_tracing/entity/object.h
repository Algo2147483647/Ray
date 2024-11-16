#ifndef RAY_TRACING_OBJECT_H
#define RAY_TRACING_OBJECT_H

#include "../properties/material.h"
#include "../properties/shape/shape.h"

using namespace Eigen;
using namespace std;

namespace RayTracing {
	struct Object {
		Shape* shape;
		Material* material;
	};
}
#endif