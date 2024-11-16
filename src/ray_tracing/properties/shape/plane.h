#ifndef RAY_TRACING_SHAPE_PLANE_H
#define RAY_TRACING_SHAPE_PLANE_H

#include "shape.h"

class Plane : public Shape {
public:
    float A, B, C, D;

    float intersect(const Vector3f& raySt, const Vector3f& ray) override {
        float t = A * ray(0) + B * ray(1) + C * ray(2);
        if (t < 0)
            return FLT_MAX;

        float d = -(A * raySt(0) + B * raySt(1) + C * raySt(2) + D) / t;
        return d > 0 ? d : FLT_MAX;
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