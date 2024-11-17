#ifndef GEOMETRY_CONVEX_HULL_H
#define GEOMETRY_CONVEX_HULL_H

#include <stack>
#include <vector>
#include <algorithm>
#include <Eigen/Dense>

using namespace std;
using namespace Eigen;

namespace Geometry {
	typedef Vector2d Point;

	vector<Point> ConvexHull(vector<Point>& points) {
		vector<Point> res;

		int n = points.size();
		if (n <= 3) return points;

		// Find the bottommost point
		Point minn = points[0];
		int minCur = 0;
		for (int i = 1; i < n; i++) {
			if (points[i][1] < minn[1] || (points[i][1] == minn[1] && points[i][0] < minn[0])) {
				minn = points[i];
				minCur = i;
			}
		}
		swap(points[0], points[minCur]);

		// Sort based on polar angle
        sort(points.begin() + 1, points.end(), [&minn](const Point& a, const Point& b) {
            int t = ((a[0] - minn[0]) * (b[1] - minn[1])) - 
                    ((b[0] - minn[0]) * (a[1] - minn[1]));

            if (t < 0)
                return true;
            if (t == 0 && (a - minn).squaredNorm() < (b - minn).squaredNorm())
                return true;
            return false;
        });

		// Construct convex hull
		vector<Point> res;
		res.push_back(points[0]);
		res.push_back(points[1]);

		for (int i = 2; i < n; i++) {
			res.push_back(points[i]);

			int m = res.size();
			while (m > 2) {
				// cross product
				int t = (res[m - 1][0] - res[m - 2][0]) * (res[m - 3][1] - res[m - 2][1]) - 
					    (res[m - 3][0] - res[m - 2][0]) * (res[m - 1][1] - res[m - 2][1]);

				if (t > 0)
					res.erase(res.begin() + m - 2);
				else break;
				m = res.size();
			}
		}

		return res;
	}

}


#endif