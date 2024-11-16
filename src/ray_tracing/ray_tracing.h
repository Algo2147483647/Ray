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
    static mutex _mutex;

    inline Vector3f TraceRay(ObjectTree& objTree, Ray& ray, int level) {
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

        bool terminate = obj->material->DielectricSurfacePropagation(ray, normalVector);
        if (terminate) {
            return ray.color;
        }

        return TraceRay(objTree, ray, level + 1);
    }

    inline void TraceRayThread(Camera& camera, ObjectTree& objTree, vector<MatrixXf>& img, vector<Ray>& rays) {
        for (auto& ray : rays) {
            Vector3f color = TraceRay(objTree, ray, 0);
            pair<int, int> coordinates = camera.GetRayCoordinates(ray, img[0].cols(), img[0].rows());

            unique_lock<mutex> lock(_mutex);
            for (int c = 0; c < 3; ++c) {
                img[c](coordinates.first, coordinates.second) += color[c];
            }
        }
    }

    inline void TraceRay(Camera& camera, ObjectTree& objTree, vector<MatrixXf>& img, int sampleSt = 0, int sampleEd = 1) {
        vector<Ray> rays = camera.GetRays(img[0].cols(), img[0].rows());
        int raysPerThread = rays.size() / threadNum;

        ThreadPool pool(threadNum);
        vector<future<void>> futures;

        for (int sample = sampleSt; sample < sampleEd; ++sample) {
            for (int i = 0; i < threadNum; ++i) {
                futures.push_back(pool.enqueue([&, i] {
                    vector<Ray> raysTmp(rays.begin() + i * threadNum, rays.begin() + (i + 1) * threadNum);
                    TraceRayThread(camera, objTree, img, raysTmp);
                }));
            }

            for (auto& future : futures) {
                future.wait();
            }
        }

        for (int i = 0; i < 3; ++i) {
            for (auto it = img[i].data(); it != img[i].data() + img[i].size(); ++it) {
                *it *= 1.0f / (sampleEd - sampleSt);
            }
        }
    }
}

#endif
