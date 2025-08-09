#ifndef RAY_TRACING_Shape_Cuboid_h
#define RAY_TRACING_Shape_Cuboid_h

#include "shape.h"

class Cuboid : public Shape {
public:
    Vector3f pmin, pmax;

    Cuboid() { ; }
    Cuboid(Vector3f pmin, Vector3f pmax) : pmin(pmin), pmax(pmax) { ; }

    string GetName() {
        return "Cuboid";
    }

    float Intersect(const Vector3f& raySt, const Vector3f& ray) override {
        thread_local Vector3f tmin_t, tmax_t;
        tmin_t = (pmin - raySt).cwiseQuotient(ray);
        tmax_t = (pmax - raySt).cwiseQuotient(ray);

        thread_local float tmin, tmax;
        tmin = tmin_t.cwiseMin(tmax_t).maxCoeff();
        tmax = tmin_t.cwiseMax(tmax_t).minCoeff();

        if (tmin > tmax || tmax < 0)
            return FLT_MAX;
        return tmin >= 0 ? tmin : tmax;
    }

    Vector3f& GetNormalVector(const Vector3f& intersect, Vector3f& res) override {
        if (fabs(intersect[0] - pmin[0]) < EPS)
            res = { -1, 0, 0 };
        else if (fabs(intersect[0] - pmax[0]) < EPS)
            res = { 1, 0, 0 };
        else if (fabs(intersect[1] - pmin[1]) < EPS)
            res = { 0, -1, 0 };
        else if (fabs(intersect[1] - pmax[1]) < EPS)
            res = { 0, 1, 0 };
        else if (fabs(intersect[2] - pmin[2]) < EPS)
            res = { 0, 0, -1 };
        else if (fabs(intersect[2] - pmax[2]) < EPS)
            res = { 0, 0, 1 };

        return res;
    }

    void BuildBoundingBox(Vector3f& pmax, Vector3f& pmin) override {
        pmax = this->pmax;
        pmin = this->pmin;
    }
};

#endif