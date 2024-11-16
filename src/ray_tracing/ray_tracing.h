#ifndef RAY_TRACING_H
#define RAY_TRACING_H

#include <vector>
#include <algorithm>
#include <thread>
#include <Eigen/Dense>
#include "utils/image.h"
#include "utils/thread_pool.h"
#include "entity/object_tree.h"
#include "entity/camera.h"
#include "optics/geometrical_optics.h"

using namespace Eigen;



namespace RayTracing {
	static int maxRayLevel = 6;
	static mutex _mutex;
	static Image imgXY, imgYZ;
	static int threadNum = 20;
	static bool is_debug = 0;


	inline Vector3f& TraceRay(const ObjectTree& objTree, Ray& ray, const int level) {
		if (level > maxRayLevel) {
			return ray.color = Vector3f(0, 0, 0);
		}

		Object* obj;
		float dis = objTree.seekIntersection(ray.origin, ray.direct, obj);
		if (dis == FLT_MAX) {
			return ray.color = Vector3f(0, 0, 0);
		}

		if (is_debug && level == 2) {
			thread_local Vector3f rayStNew;
			rayStNew = ray.origin + dis * ray.direct;
			unique_lock<mutex> lock(_mutex);
			Graphics::drawLine(imgXY, rayStNew[0], rayStNew[1], ray.origin[0], ray.origin[1]);
			Graphics::drawLine(imgYZ, rayStNew[1], rayStNew[2], ray.origin[1], ray.origin[2]);
		}

		if (obj->material->rediate) {
			ray.color = ray.color.cwiseProduct(obj->material->baseColor);
			return ray.color;
		}
		
		ray.origin += dis * ray.direct;
		thread_local Vector3f faceVec;
		{
			obj->shape->faceVector(ray.origin, faceVec);
			if (faceVec.dot(ray.direct) > 0) {
				faceVec *= -1;
			}
		}
		
		obj->material->dielectricSurfacePropagation(ray, faceVec);
		traceRay(objTree, ray, level + 1);
		return ray.color;
	}

	 
	inline void TraceRayThread(const Camera& camera, const ObjectTree& objTree, vector<MatrixXf>& img, int xSt, int xEd, int ySt, int yEd) {

		thread_local Vector3f sampleVec;
		thread_local Ray ray;
		
		for (int y = ySt; y < yEd; y++) {
			for (int x = xSt; x < xEd; x++) {
				sampleVec = (x + distribution(gen) - img[0].rows() / 2.0 - 0.5) * camera.ScreenXVec +
					        (y + distribution(gen) - img[0].cols() / 2.0 - 0.5) * camera.ScreenYVec;
				ray.origin =  camera.center + sampleVec;
				ray.direct = (camera.direct + sampleVec).normalized();
				ray.color = Vector3f::Ones();

				traceRay(objTree, ray, 0);

				{
					unique_lock<mutex> lock(_mutex);
					for (int c = 0; c < 3; c++) 
						img[c](x, y) += ray.color[c];
				}
			}
		}
	}

	inline void TraceRay(const Camera& camera, const ObjectTree& objTree, vector<MatrixXf>& img, int sampleSt, int sampleEd) {
		int threadSize = img[0].rows() / threadNum;
		ThreadPool pool(threadNum);
		vector<future<void>> futures;
		
		for (int sample = sampleSt; sample < sampleEd; sample++) {
			for (int i = 0; i < threadNum; i++) {
				futures.push_back(pool.enqueue([&, i] {
					TraceRayThread(camera, objTree, img, i * threadSize, (i + 1) * threadSize, 0, img[0].cols());
				}));
			}
		}

		// Wait for all tasks to complete
		for (auto& future : futures) {
			future.wait();
		}

		for(int i = 0; i < 3; i++) {
			img[i] *= 1.0 / (sampleEd - sampleSt);
		}
	}
}

#endif