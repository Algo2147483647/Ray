#ifndef INTERSECT_H
#define INTERSECT_H

#include <float.h>
#include <algorithm>
#include <complex>
#include <cmath>
#include <Eigen/Dense>
#include <corecrt_math_defines.h>

using namespace std;
using namespace Eigen;

constexpr float EPS = 10e-4;

namespace Intersect {
	/*#############################################################################
	*
	*						求交点
	*
	##############################################################################*/
	/*
	 * 线段、线段交点
	 */
	bool OnSegments(Vector3f& p1, Vector3f& p2, Vector3f& p3) {
		if (min(p1(1), p2(1)) <= p3(1) && 
			max(p1(1), p2(1)) >= p3(1) &&
			min(p1(2), p2(2)) <= p3(2) && 
			max(p1(2), p2(2)) >= p3(2) )
			return true;
		return false;
	}

	bool Segments(Vector3f& p1, Vector3f& p2, Vector3f& p3, Vector3f& p4) {
		float
			d1 = (p1(1) - p3(1)) * (p4(2) - p3(2)) - (p4(1) - p3(1)) * (p1(2) - p3(2)),
			d2 = (p2(1) - p3(1)) * (p4(2) - p3(2)) - (p4(1) - p3(1)) * (p2(2) - p3(2)),
			d3 = (p3(1) - p1(1)) * (p2(2) - p1(2)) - (p2(1) - p1(1)) * (p3(2) - p1(2)),
			d4 = (p4(1) - p1(1)) * (p2(2) - p1(2)) - (p2(1) - p1(1)) * (p4(2) - p1(2));

		if (((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && 
			((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)))
			return true;

		else if (d1 == 0 && OnSegments(p3, p4, p1))
			return true;

		else if (d2 == 0 && OnSegments(p3, p4, p2))
			return true;

		else if (d3 == 0 && OnSegments(p1, p2, p3))
			return true;

		else if (d4 == 0 && OnSegments(p1, p2, p4))
			return true;

		return false;
	}

}

#endif