#ifndef COMPUTATIONAL_GEOMETRY_DELAUNAY_H
#define COMPUTATIONAL_GEOMETRY_DELAUNAY_H

#include <algorithm>
#include <vector>

using namespace std;

namespace Geometry {
	typedef vector<float> Point;

	vector<Point> Delaunay(vector<Point>& points) {
		vector<Point> triAns, triTemp, edgeBuffer;
		sort(points.begin(), points.end(), [](Point& a, Point& b) {				// 将点按坐标x从小到大排序
			return a[0] != b[0] ? a[0] < b[0] : a[1] < b[1];
			});

		//[2]
		Point maxPoint(points[0]),
			  minPoint(points[0]);
		for (int i = 1; i < points.size(); i++) {
			maxPoint = (points[i][0] > maxPoint[0] || (points[i][0] == maxPoint[0] && points[i][1] > maxPoint[1])) ? points[i] : maxPoint;
			minPoint = (points[i][0] < minPoint[0] || (points[i][0] == minPoint[0] && points[i][1] < minPoint[1])) ? points[i] : minPoint;
		}
		Point supertriangle(2, 3);
		Point length = maxPoint - minPoint;
		supertriangle(0, 0) = minPoint[0] - length[0] - 2;    
		supertriangle(0, 1) = maxPoint[0] + length[0] + 2;   
		supertriangle(0, 2) = (maxPoint[0] + minPoint[0]) / 2; 
		supertriangle(1, 0) = minPoint[1] - 2;
		supertriangle(1, 1) = minPoint[1] - 2;
		supertriangle(1, 2) = maxPoint[1] + length[1] + 2;
		triTemp.push_back(supertriangle);
		//[3]
		for (int i = 0; i < points.size(); i++) {
			edgeBuffer.clear();

			for (int j = 0; j < triTemp.size(); j++) {

				Point center, triEdge[3], temp;
				for (int k = 0; k < 3; k++)
					getCol(triTemp[j], k, triEdge[k]);
				double R;
				ThreePoints2Circle(triEdge, center, R);
				double distance = (sub(temp, point[i], center)).norm();

				if (point[i][0] > center[0] + R) {
					triAns.push_back(triTemp[j]);
					triTemp.erase(triTemp.begin() + j--);
				}

				else if (distance < R) {
					Point edge(2, 2), p1, p2;
					for (int k = 0; k < 3; k++) {
						getCol(triTemp[j], k, p1);
						getCol(triTemp[j], k + 1) % 3, p2);
						if (p1[0] < p2[0] || (p1[0] == p2[0] && p1[1] < p2[1])) {
							setCol(edge, 0, p1);
							setCol(edge, 1, p2);
						}
						else {
							setCol(edge, 0, p2);
							setCol(edge, 1, p1);
						}
						edgeBuffer.push_back(edge);
					}
					triTemp.erase(triTemp.begin() + j--);
				}
			}

			sort(edgeBuffer.begin(), edgeBuffer.end(), [](Point a, Point b) {
				if (a(0, 0) < b(0, 0) || (a(0, 0) == b(0, 0) && a(1, 0) < b(1, 0)))return true;
			if (a(0, 1) < b(0, 1) || (a(0, 1) == b(0, 1) && a(1, 1) < b(1, 1)))return true;
			return false;
				});
			for (int j = 0; j < edgeBuffer.size() - 1; j++) {
				bool flag = 0;
				while (j + 1 < edgeBuffer.size() && edgeBuffer[j] == edgeBuffer[j + 1]) {
					edgeBuffer.erase(edgeBuffer.begin() + j + 1); flag = 1;
				}
				if (flag) { edgeBuffer.erase(edgeBuffer.begin() + j); j--; }
			}
			//[3.4] 
			for (int j = 0; j < edgeBuffer.size(); j++) {
				Point t(2, 3), temp;
				t.setCol(0, edgeBuffer[j].getCol(0, temp));
				t.setCol(1, edgeBuffer[j].getCol(1, temp));
				t.setCol(2, point[i]);
				triTemp.push_back(t);
			}
		}
		//[4]
		for (int i = 0; i < triTemp.size(); i++) triAns.push_back(triTemp[i]);
		for (int i = 0; i < triAns.size(); i++) {
			Point t;
			for (int j = 0; j < 3; j++) {
				triAns[i].getCol(j, t);
				if (t[0]< minPoint[0] || t[1] < minPoint[1] || t[0] > maxPoint[0] || t[1] > maxPoint[1]) {
					triAns.erase(triAns.begin() + i--); break;
				}
			}
		}
		// [Output]
		TrianglesNum = triAns.size();
		Point* Triangles = (Point*)calloc(TrianglesNum, sizeof(Point));
		for (int i = 0; i < TrianglesNum; i++) Triangles[i] = triAns[i];
		return Triangles;
	}


}
#endif