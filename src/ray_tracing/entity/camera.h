#ifndef RAY_TRACING_CAMERA_H
#define RAY_TRACING_CAMERA_H

#include <corecrt_math_defines.h>
#include <Eigen/Dense>
#include "../utils/consts.h"

using namespace Eigen;

namespace RayTracing {

    struct Camera {
        Vector3f position;   // Camera position in the scene
        Vector3f lookAt;     // Point the camera is looking at
        Vector3f up;         // Up vector to define the camera orientation
        float fieldOfView;   // Field of view in degrees
        float aspectRatio;   // Aspect ratio of the image (width / height)

        Camera(const Vector3f& pos, const Vector3f& look, const Vector3f& upVec,
            float fov, float aspect)
            : position(pos), lookAt(look), up(upVec), fieldOfView(fov), aspectRatio(aspect) {}

        std::vector<Ray> GetRays(int width, int height) const {
            std::vector<Ray> rays;
            rays.reserve(width * height);

            Vector3f forward = (lookAt - position).normalized();
            Vector3f right = forward.cross(up).normalized();
            Vector3f cameraUp = right.cross(forward).normalized();

            // Calculate the dimensions of the image plane
            float imageHeight = 2 * tan(fieldOfView * 0.5 * M_PI / 180.0);
            float imageWidth = imageHeight * aspectRatio;

            // Loop through each pixel and generate rays
            for (int j = 0; j < height; ++j) {
                for (int i = 0; i < width; ++i) {
                    // Add random offsets for Monte Carlo sampling
                    float randX = distribution(generator);
                    float randY = distribution(generator);

                    // Calculate normalized device coordinates with randomness
                    float u = (2 * ((i + randX) / width) - 1) * imageWidth / 2;
                    float v = (1 - 2 * ((j + randY) / height)) * imageHeight / 2;


                    // Calculate the ray direction
                    Vector3f rayDirection = (forward + u * right + v * cameraUp).normalized();
                    rays.emplace_back(position, rayDirection);
                }
            }

            return rays;
        }

    };
}

#endif