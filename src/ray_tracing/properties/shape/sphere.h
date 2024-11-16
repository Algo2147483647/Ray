#ifndef RAY_TRACING_Shape_Sphere_h
#define RAY_TRACING_Shape_Sphere_h

#include "shape.h"

class Sphere : public Shape {
public:
    Vector3f center;
    float R = 0;

    Sphere() { ; }
    Sphere(Vector3f center, float R) : center(center), R(R) {; }
    Sphere(Vector3f center, float R, std::function<bool(Vector3f&)> engraving) : center(center), R(R), Shape(engraving){; }

    string GetName() {
        return "Sphere";
    }

    float Intersect(const Vector3f& raySt, const Vector3f& ray) override {
        thread_local Vector3f rayStCenter;
        rayStCenter = raySt - center;

        float
            A = ray.dot(ray),
            B = 2 * ray.dot(rayStCenter),
            C = rayStCenter.dot(rayStCenter) - R * R,
            Delta = B * B - 4 * A * C;

        if (Delta < 0)
            return FLT_MAX;									//有无交点
        Delta = sqrt(Delta);

        float root1 = (-B - Delta) / (2 * A);
        float root2 = (-B + Delta) / (2 * A);

        if (engraving != nullptr) {
            thread_local Vector3f intersection;

            if (root1 > 0) {
                intersection = (raySt + root1 * ray - center).normalized();
                if (engraving(intersection)) {
                    return root1;
                }
            }
            if (root2 > 0) {
                intersection = (raySt + root2 * ray - center).normalized();
                if (engraving(intersection)) {
                    return root2;
                }
            }
            return FLT_MAX;
        }

        if (root1 > 0 && root2 > 0) return std::min(root1, root2);
        if (root1 > 0 || root2 > 0) return std::max(root1, root2);
        return FLT_MAX;
    }

    Vector3f& GetNormalVector(const Vector3f& intersect, Vector3f& res) override {
        res = intersect - center;
        res.normalize();
        return res;
    }

    void BuildBoundingBox(Vector3f& pmax, Vector3f& pmin) override {
        pmax = center + Vector3f(R, R, R);
        pmin = center - Vector3f(R, R, R);
    }
};


#endif