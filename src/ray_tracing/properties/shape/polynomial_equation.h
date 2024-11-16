#ifndef RAY_TRACING_SHAPE_PolynomialEquation_H
#define RAY_TRACING_SHAPE_PolynomialEquation_H

#include "shape.h"
#include <cmath>
#include <Eigen/Dense>
#include <vector>

using namespace Eigen;
using namespace std;

class PolynomialEquation : public Shape {
public:
    MatrixXf coeffs; // Coefficients for the polynomial equation

    string GetName() override {
        return "PolynomialEquation";
    }

    float Intersect(const Vector3f& raySt, const Vector3f& ray) override {
        // Implement ray-polygon intersection (e.g., solving F(x, y, z) = 0)
        // Assuming a single variable polynomial function F(x, y, z) = 0:
        // Replace this with your specific polynomial equation logic.

        // Example: Simple quadratic surface F(x, y, z) = ax^2 + by^2 + cz^2 - d
        float a = coeffs(0, 0);
        float b = coeffs(1, 0);
        float c = coeffs(2, 0);
        float d = coeffs(3, 0);

        float A = a * ray.x() * ray.x() + b * ray.y() * ray.y() + c * ray.z() * ray.z();
        float B = 2 * (a * raySt.x() * ray.x() + b * raySt.y() * ray.y() + c * raySt.z() * ray.z());
        float C = a * raySt.x() * raySt.x() + b * raySt.y() * raySt.y() + c * raySt.z() * raySt.z() - d;

        // Solve quadratic equation: At^2 + Bt + C = 0
        float discriminant = B * B - 4 * A * C;
        if (discriminant < 0) {
            return FLT_MAX; // No intersection
        }

        float t1 = (-B - sqrt(discriminant)) / (2 * A);
        float t2 = (-B + sqrt(discriminant)) / (2 * A);

        if (t1 > 0 && t1 < t2) {
            return t1;
        }
        else if (t2 > 0) {
            return t2;
        }

        return FLT_MAX; // No valid intersection
    }

    Vector3f& GetNormalVector(const Vector3f& intersect, Vector3f& res) override {
        // Compute gradient of the polynomial equation at the intersection point
        // Example for F(x, y, z) = ax^2 + by^2 + cz^2 - d:
        float a = coeffs(0, 0);
        float b = coeffs(1, 0);
        float c = coeffs(2, 0);

        res.x() = 2 * a * intersect.x();
        res.y() = 2 * b * intersect.y();
        res.z() = 2 * c * intersect.z();
        res.normalize(); // Normalize the result to get a unit vector
        return res;
    }

    void GetBoundingBox(Vector3f& pmax, Vector3f& pmin) override {
        // Optionally compute bounding box for the polynomial shape
        pmin = Vector3f(-FLT_MAX, -FLT_MAX, -FLT_MAX);
        pmax = Vector3f(FLT_MAX, FLT_MAX, FLT_MAX);
    }
};

#endif
