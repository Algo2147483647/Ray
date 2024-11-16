#ifndef GEOMETERICAL_OPTICS_H
#define GEOMETERICAL_OPTICS_H

#include <Eigen/Dense>
#include <random>
#include <corecrt_math_defines.h>

using namespace Eigen;

namespace GeometricalOptics {
    inline float computeHaze(float intensity, float ambientLight, float distance, float attenuationCoefficient) {
        float transmissionFactor = exp(-attenuationCoefficient * distance);
        return intensity * transmissionFactor + ambientLight * (1 - transmissionFactor);
    }

    inline Vector3f computeHaze(const Vector3f& intensity, const Vector3f& ambientLight, float distance, float attenuationCoefficient) {
        float transmissionFactor = exp(-attenuationCoefficient * distance);
        return transmissionFactor * intensity + (1 - transmissionFactor) * ambientLight;
    }

}

#endif