#ifndef RAY_TRACING_SHAPE_H
#define RAY_TRACING_SHAPE_H

#include <vector>
#include <memory>
#include <string>
#include <algorithm>
#include <Eigen/Dense>
#include "consts.h"
#include "Graphics2D.h"

using namespace Eigen;
using namespace std;

// Base Shape class
class Shape {
public:
    std::function<bool(Vector3f&)> engraving = nullptr;

    Shape() { ; }
    Shape(std::function<bool(Vector3f&)> engraving) : engraving(engraving) { ; }

    virtual string GetName();
    virtual float intersect(const Vector3f& raySt, const Vector3f& ray) = 0;  // Pure virtual function
    virtual Vector3f& GetNormalVector(const Vector3f& intersect, Vector3f& res) = 0;
    virtual void boundingBox(Vector3f& pmax, Vector3f& pmin) = 0;
    virtual void paint(Image& imgXY, Image& imgYZ) = 0;
};

#endif