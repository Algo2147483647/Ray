#ifndef RAY_TRACING_SHAPE_H
#define RAY_TRACING_SHAPE_H

#include <vector>
#include <memory>
#include <string>
#include <algorithm>
#include <Eigen/Dense>
#include "../../utils/consts.h"

using namespace Eigen;
using namespace std;

// Base Shape class
class Shape {
public:
    std::function<bool(Vector3f&)> engraving = nullptr;

    Shape() { ; }
    Shape(std::function<bool(Vector3f&)> engraving) : engraving(engraving) { ; }

    virtual string GetName() = 0;
    virtual float Intersect(const Vector3f& raySt, const Vector3f& ray) = 0;
    virtual Vector3f& GetNormalVector(const Vector3f& intersect, Vector3f& res) = 0;
    virtual void BuildBoundingBox(Vector3f& pmax, Vector3f& pmin) = 0;
};

#endif