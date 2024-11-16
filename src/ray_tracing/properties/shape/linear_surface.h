#ifndef RAY_TRACING_Shape_LinearSurface_h
#define RAY_TRACING_Shape_LinearSurface_h

#include "shape.h"

class LinearSurface : public Shape {
public:
    int order = 1;

    float intersect(const Vector3f& raySt, const Vector3f& ray) override {
        return FLT_MAX;
    }

    Vector3f& GetNormalVector(const Vector3f& intersect, Vector3f& res) override {
        return res;
    }

    void boundingBox(Vector3f& pmax, Vector3f& pmin) override {
        ;
    }

    void paint(Image& imgXY, Image& imgYZ) override {
        ;
    }
};

#endif