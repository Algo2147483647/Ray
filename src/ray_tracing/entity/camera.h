#ifndef RAY_TRACING_CAMERA_H
#define RAY_TRACING_CAMERA_H

#include <corecrt_math_defines.h>
#include <Eigen/Dense>

using namespace Eigen;

namespace RayTracing {

    struct Camera {
        Vector3f position;   // Camera position in the scene
        Vector3f lookAt;     // Point the camera is looking at
        Vector3f up;         // Up vector to define the camera orientation
        float fieldOfView;   // Field of view in degrees
        float aspectRatio;   // Aspect ratio of the image (width / height)
        float nearPlane;     // Distance to the near clipping plane
        float farPlane;      // Distance to the far clipping plane

        // Constructor
        Camera(const Vector3f& pos, const Vector3f& look, const Vector3f& upVec,
            float fov, float aspect, float nearP = 0.1f, float farP = 1000.0f)
            : position(pos), lookAt(look), up(upVec), fieldOfView(fov), aspectRatio(aspect),
            nearPlane(nearP), farPlane(farP) {}

        // Method to calculate the view matrix
        Matrix4f viewMatrix() const {
            Vector3f zAxis = (position - lookAt).normalized(); // Forward
            Vector3f xAxis = up.cross(zAxis).normalized();    // Right
            Vector3f yAxis = zAxis.cross(xAxis);              // Up

            Matrix4f view = Matrix4f::Identity();
            view.block<3, 1>(0, 0) = xAxis;
            view.block<3, 1>(0, 1) = yAxis;
            view.block<3, 1>(0, 2) = zAxis;
            view.block<3, 1>(0, 3) = -position;

            return view;
        }

        // Method to calculate the projection matrix
        Matrix4f projectionMatrix() const {
            float tanHalfFov = tan(fieldOfView * M_PI / 360.0f);
            Matrix4f projection = Matrix4f::Zero();
            projection(0, 0) = 1.0f / (aspectRatio * tanHalfFov);
            projection(1, 1) = 1.0f / tanHalfFov;
            projection(2, 2) = -(farPlane + nearPlane) / (farPlane - nearPlane);
            projection(2, 3) = -2.0f * farPlane * nearPlane / (farPlane - nearPlane);
            projection(3, 2) = -1.0f;
            return projection;
        }
    };
}

#endif