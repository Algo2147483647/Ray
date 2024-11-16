#ifndef RAY_TRACING_Shape_Triangle_h
#define RAY_TRACING_Shape_Triangle_h

#include "shape.h"

class Triangle : public Shape {
public:
    Vector3f p1, p2, p3;

    Triangle() { ; }
    Triangle(Vector3f p1, Vector3f p2, Vector3f p3) : p1(p1), p2(p2), p3(p3) { ; }

    float intersect(const Vector3f& raySt, const Vector3f& ray) override {
        thread_local Vector3f edge[2], tmp, p, q;
        edge[0] = p2 - p1;
        edge[1] = p3 - p1;

        // p & a & tmp
        p = ray.cross(edge[1]);
        float a = edge[0].dot(p);

        if (a > 0)
            tmp = raySt - p1;
        else {
            tmp = p1 - raySt;
            a = -a;
        }

        if (a < EPS)
            return FLT_MAX;								//射线与三角面平行

        // u & q & v
        float u = tmp.dot(p) / a;
        if (u < 0 || u > 1)
            return FLT_MAX;

        q = tmp.cross(edge[0]);
        float v = q.dot(ray) / a;
        if (v < 0 || u + v > 1)
            return FLT_MAX;

        return q.dot(edge[1]) / a;
    }

    Vector3f& faceVector(const Vector3f& intersect, Vector3f& res) override {
        res = (p2 - p1)
              .cross(p3 - p1)
              .normalized();
        return res;
    }

    void boundingBox(Vector3f& pmax, Vector3f& pmin) override {
        for (int j = 0; j < 3; j++) {
            pmin[j] = min(p1[j], min(p2[j], p3[j]));
            pmax[j] = max(p1[j], max(p2[j], p3[j]));
        }
    }

    void paint(Image& imgXY, Image& imgYZ) override {
        Graphics::drawTriangle(imgXY, p1[0], p1[1], p2[0], p2[1], p3[0], p3[1]);
        Graphics::drawTriangle(imgYZ, p1[1], p1[2], p2[1], p2[2], p3[1], p3[2]);
    }
};

#endif