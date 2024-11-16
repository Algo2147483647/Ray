#ifndef RAY_TRACING_CAMERA_H
#define RAY_TRACING_CAMERA_H

#include <vector>
#include <corecrt_math_defines.h>
#include <Eigen/Dense>
#include "../utils/consts.h"

using namespace Eigen;
using namespace std;

namespace RayTracing {

    struct Camera {
        Vector3f position;   // Camera position in the scene
        Vector3f direction;     // Point the camera is looking at
        Vector3f up;         // Up vector to define the camera orientation
        float fieldOfView;   // Field of view in degrees
        float aspectRatio;   // Aspect ratio of the image (width / height)

        Camera() { ; }

        void SetLookAt(Vector3f& lookAt) {
            direction = (lookAt - position).normalized();
        }

        vector<Ray> GetRays(int width, int height) const {
            vector<Ray> rays;
            rays.reserve(width * height);

            Vector3f right = direction.cross(up).normalized();
            Vector3f cameraUp = right.cross(direction).normalized();

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
                    Vector3f rayDirection = (direction + u * right + v * cameraUp).normalized();
                    rays.emplace_back(position, rayDirection);
                }
            }
            return rays;
        }

        pair<int, int> GetRayCoordinates(const Ray& ray, int width, int height) const {
            Vector3f right = direction.cross(up).normalized();
            Vector3f cameraUp = right.cross(direction).normalized();

            // Calculate the dimensions of the image plane
            float imageHeight = 2 * tan(fieldOfView * 0.5 * M_PI / 180.0);
            float imageWidth = imageHeight * aspectRatio;

            // Reverse mapping from ray direction to pixel coordinates
            Vector3f dir = ray.direction.normalized();
            float u = dir.dot(right) / (imageWidth / 2);
            float v = dir.dot(cameraUp) / (imageHeight / 2);

            // Convert normalized device coordinates to pixel coordinates
            int i = static_cast<int>((u + 1) / 2 * width);
            int j = static_cast<int>((1 - v) / 2 * height);

            return make_pair(i, j);
        }
    };
}

#endif