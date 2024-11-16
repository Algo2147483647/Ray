#ifndef RAY_TRACING_H
#define RAY_TRACING_H

#include <vector>
#include <algorithm>
#include <thread>
#include <Eigen/Dense>
#include "Object.h"
#include "ThreadPool.h"

using namespace Eigen;

/*#############################################################################
*
*						光线追踪  Ray Tracing
*
##############################################################################*/

namespace RayTracing {
	static int maxRayLevel = 6;
	static mutex _mutex;
	static Image imgXY, imgYZ;
	static int threadNum = 20;
	static bool is_debug = 0;

	/******************************************************************************
	*						追踪光线
	*	[步骤]:
			[1] 遍历三角形集合中的每一个三角形
				[2]	判断光线和该三角形是否相交、光线走过距离、交点坐标、光线夹角
				[3] 保留光线走过距离最近的三角形的相关数据
			[4] 如果该光线等级小于设定的阈值等级
				计算三角形反射方向，将反射光线为基准重新计算
	&	[注]:distance > 1而不是> 0，是因为反射光线在接触面的精度内，来回碰自己....
	******************************************************************************/
	inline Vector3f& traceRay(const ObjectTree& objTree, Ray& ray, const int level) {
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

	 
	inline void traceRay_thread(const Camera& camera, const ObjectTree& objTree, vector<MatrixXf>& img, int xSt, int xEd, int ySt, int yEd) {

		thread_local Vector3f sampleVec;
		thread_local Ray ray;
		
		for (int y = ySt; y < yEd; y++) {
			for (int x = xSt; x < xEd; x++) {
				sampleVec = (x + dis(gen) - img[0].rows() / 2.0 - 0.5) * camera.ScreenXVec +
					        (y + dis(gen) - img[0].cols() / 2.0 - 0.5) * camera.ScreenYVec;
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

	inline void traceRay(const Camera& camera, const ObjectTree& objTree, vector<MatrixXf>& img, int sampleSt, int sampleEd) {
		int threadSize = img[0].rows() / threadNum;
		ThreadPool pool(threadNum);
		vector<future<void>> futures;
		
		for (int sample = sampleSt; sample < sampleEd; sample++) {
			for (int i = 0; i < threadNum; i++) {
				futures.push_back(pool.enqueue([&, i] {
					traceRay_thread(camera, objTree, img, i * threadSize, (i + 1) * threadSize, 0, img[0].cols());
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

	inline void debug(const Camera& camera, const ObjectTree& objTree) {
		Graphics::PaintSize = 1;
		imgXY = Image(3000, 3000);
		imgYZ = Image(3000, 3000);
		imgXY.setZero();
		imgYZ.setZero();

		Graphics::PaintColor = 0xFFFFFF;
		for (auto& obj : objTree.ObjectSet) {
			obj.shape->paint(imgXY, imgYZ);
		}

		{
			Graphics::PaintColor = 0xFF0000;
			Graphics::drawLine(imgXY, camera.center[0],  camera.center[1],
				                      camera.center[0] + camera.direct[0],
				                      camera.center[1] + camera.direct[1]);
			Graphics::drawLine(imgYZ, camera.center[1],  camera.center[2],
				                      camera.center[1] + camera.direct[1],
				                      camera.center[2] + camera.direct[2]);

			Graphics::PaintColor = 0x00FF00;
			Graphics::drawLine(imgXY, camera.center[0],  camera.center[1],
				                      camera.center[0] + camera.ScreenYVec[0] * 100, 
				                      camera.center[1] + camera.ScreenYVec[1] * 100);

			Graphics::PaintColor = 0x0000FF;
			Graphics::drawLine(imgYZ, camera.center[1],  camera.center[2],
				                      camera.center[1] + camera.ScreenXVec[1] * 100,
				                      camera.center[2] + camera.ScreenXVec[2] * 100);

			Graphics::PaintColor = 0xFFFF00;
			Graphics::PaintSize = 0;
		}

		Graphics::ppmWrite("C:/Users/29753/Desktop/outXY.ppm", imgXY);
		Graphics::ppmWrite("C:/Users/29753/Desktop/outYZ.ppm", imgYZ);
	}
}

#endif