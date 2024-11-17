#ifndef MODELING_H
#define MODELING_H

#include <vector>
#include <corecrt_math_defines.h>
#include <Eigen/Dense>
#include "GraphicsIO.h"
#include "MarchingCubes.h"
#include "Ear_Clipping.h"

using namespace std;
using namespace Eigen;

namespace Modeling {
	typedef Vector3f Point;   // x, y, z
	typedef vector<float> triangle;  // p1, p2, p3

	class Mesh {
	public:
		vector<triangle> Object;

		inline Mesh& operator= (Mesh& a) {
			Object = a.Object;
			return *this;
		}

		void write(const string fileName);
	};


	/* 图形 */
	void Rotator(Point& center, Point& axis, vector<Point>& f, int pointNum, float st = 0, float ed = 2 * M_PI, int isClosed = false);		// 旋转体 
	void Translator(Point& st, Point& ed, vector<Point>& f, int isClosed = true);			// 平移体 
	void Translator(vector<Point>& path, vector<Point>& f, int isClosed = true);
	void Rotator_Translator(Point& center, Point& axis, vector<Point>& f, Vector3f& direction, float length, int pointNum, float st = 0, float ed = 2 * M_PI);

	/* 2D Points */
	inline vector<Point> Circle(float R, int N, int clockwise = -1) {
		vector<Point> res;

		for (int i = 0; i <= N; i++) {
			float angle = clockwise * i / (float)N * 2 * M_PI;
			res.push_back({ R * sin(angle), R * cos(angle) });
		}
		return res;
	}

	/* 2D Modeling */
	void Triangle(Point& p1, Point& p2, Point& p3);
	void Rectangle(Point& c, float X, float Y);
	void Quadrangle(Point& p1, Point& p2, Point& p3, Point& p4);
	void ConvexPolygon(vector<Point>& p);
	void Polygon(Point& c, vector<Point>& p);
	void Circle(Point& center, float r, int pointNum, float angleSt = 0, float angleEd = 2 * M_PI);
	void Surface(Mat<float>& z, float xs, float xe, float ys, float ye, Point* direct = nullptr);

	/* 3D Modeling */
	void Tetrahedron(Point& p1, Point& p2, Point& p3, Point& p4);
	void Cuboid(Point& pMin, Point& pMax);
	void Cuboid(Point& center, float X, float Y, float Z);
	void Cuboid(Point& center, Vector3f& direction, float L, float W, float H);
	void Frustum(Point& st, Point& ed, float Rst, float Red, int pointNum);							// 画圆台
	void Sphere(Point& center, float r, int pointNum);
	void Sphere(Point& center, float r, int ThetaNum, int PhiNum,
		float thetaSt = 0, float thetaEd = 2 * M_PI,
		float phiSt = -M_PI / 2, float phiEd = M_PI / 2);

	/* Modifier */
	void addTriangleSet(Point& center, vector<triangle>& tris);
	void Array(int count, float dx, float dy, float dz);


}
#endif