#ifndef GEOMETERICAL_OPTICS_H
#define GEOMETERICAL_OPTICS_H

#include <consts>
#include <Eigen/Dense>
#include <random>
#include <corecrt_math_defines.h>

using namespace Eigen;

namespace GeometricalOptics {
	inline float Haze(float I, float A, float dis, float beta) {
		float t = exp(-beta * dis);
		return I * t + A * (1 - t);
	}

	inline Vector3f Haze(const Vector3f& I, const Vector3f& A, float dis, float beta) {
		float t = exp(-beta * dis);
		return t * I + (1 - t) * A;
	}
}

#endif