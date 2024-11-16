#ifndef RAY_TRACING_SHAPE_PLANE_H
#define RAY_TRACING_SHAPE_PLANE_H

#include "shape.h"

class Plane : public Shape {
public:
    float A, B, C, D;

    string GetName() {
        return "Plane";
    }

    float Intersect(const Vector3f& raySt, const Vector3f& ray) override {
        float t = A * ray(0) + B * ray(1) + C * ray(2);
        if (t < 0)
            return FLT_MAX;

        float d = -(A * raySt(0) + B * raySt(1) + C * raySt(2) + D) / t;
        return d > 0 ? d : FLT_MAX;
    }

    Vector3f& GetNormalVector(const Vector3f& intersect, Vector3f& res) override {
        res = Vector3f(A, B, C).normalized();
        return res;
    }

    void GetBoundingBox(Vector3f& pmax, Vector3f& pmin)  {
        // Infinite plane, so bounding box is typically not defined
        // Depending on the application, we might set extremely large values
        pmin = Vector3f(-FLT_MAX, -FLT_MAX, -FLT_MAX);
        pmax = Vector3f(FLT_MAX, FLT_MAX, FLT_MAX);
    }

};

#endif