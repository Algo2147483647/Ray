#ifndef RAY_TRACING_H
#define RAY_TRACING_H

#include <vector>
#include <algorithm>
#include <thread>
#include <mutex>
#include <Eigen/Dense>
#include "../image/image.h"
#include "utils/thread_pool.h"
#include "entity/object_tree.h"
#include "entity/camera.h"
#include "optics/geometrical_optics.h"

using namespace Eigen;

namespace RayTracing {
    static int MaxRayLevel = 6;
    static int threadNum = 20;
    static std::mutex _mutex;

    inline Vector3f TraceRay(ObjectTree& objTree, Ray& ray, const int level) {
        if (level > MaxRayLevel) {
            return Vector3f(0, 0, 0);
        }

        Object* obj = nullptr;
        float dis = objTree.GetIntersection(ray.origin, ray.direction, objTree.root, obj);
        if (dis == FLT_MAX) {
            return Vector3f(0, 0, 0);
        }

        ray.origin += dis * ray.direction;
        Vector3f normalVector;

        obj->shape->GetNormalVector(ray.origin, normalVector);
        if (normalVector.dot(ray.direction) > 0) {
            normalVector *= -1;
        }

        DielectricSurfacePropagation(*obj->material, ray, normalVector);
        return TraceRay(objTree, ray, level + 1);
    }

    inline void TraceRayThread(const Camera& camera, const ObjectTree& objTree, std::vector<MatrixXf>& img, const std::vector<Ray>& rays, int start, int end) {
        for (int idx = start; idx < end; ++idx) {
            Ray ray = rays[idx];
            Vector3f color = TraceRay(objTree, ray, 0);

            pair<int, int> coordinates = camera.GetRayCoordinates(ray, img[0].cols(), img[0].rows());

            std::unique_lock<std::mutex> lock(_mutex);
            for (int c = 0; c < 3; ++c) {
                img[c](coordinates.first, coordinates.second) += color[c];
            }
        }
    }

    inline void TraceRay(const Camera& camera, const ObjectTree& objTree, std::vector<MatrixXf>& img, int sampleSt, int sampleEd) {
        std::vector<Ray> rays = camera.GetRays(img[0].cols(), img[0].rows());
        int raysPerThread = static_cast<int>(rays.size()) / threadNum;

        ThreadPool pool(threadNum);
        std::vector<std::future<void>> futures;

        for (int sample = sampleSt; sample < sampleEd; ++sample) {
            for (int i = 0; i < threadNum; ++i) {
                int startIdx = i * raysPerThread;
                int endIdx = (i == threadNum - 1) ? static_cast<int>(rays.size()) : (i + 1) * raysPerThread;
                futures.push_back(pool.enqueue([&, startIdx, endIdx] {
                    TraceRayThread(camera, objTree, img, rays, startIdx, endIdx);
                    }));
            }

            for (auto& future : futures) {
                future.wait();
            }
        }

        for (int i = 0; i < 3; ++i) {
            img[i] *= 1.0f / (sampleEd - sampleSt);
        }
    }
}

#endif
