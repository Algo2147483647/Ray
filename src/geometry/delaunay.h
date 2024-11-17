#ifndef COMPUTATIONAL_GEOMETRY_DELAUNAY_H
#define COMPUTATIONAL_GEOMETRY_DELAUNAY_H

#include <algorithm>
#include <vector>
#include <cmath>

using namespace std;

namespace Geometry {

    using Point = vector<float>;

    // Utility function to calculate the determinant of a 3x3 matrix
    float determinant(const Point& a, const Point& b, const Point& c) {
        return a[0] * (b[1] * c[2] - b[2] * c[1]) -
            a[1] * (b[0] * c[2] - b[2] * c[0]) +
            a[2] * (b[0] * c[1] - b[1] * c[0]);
    }

    // Function to calculate the circumcenter of a triangle given by points a, b, and c
    void circumcenter(const Point& a, const Point& b, const Point& c, Point& center, float& radius) {
        float d = 2 * determinant(a, b, c);
        float ax = a[0], ay = a[1];
        float bx = b[0], by = b[1];
        float cx = c[0], cy = c[1];

        float A = ax * ax + ay * ay;
        float B = bx * bx + by * by;
        float C = cx * cx + cy * cy;

        center[0] = (A * (by - cy) + B * (cy - ay) + C * (ay - by)) / d;
        center[1] = (A * (cx - bx) + B * (ax - cx) + C * (bx - ax)) / d;
        radius = sqrt((center[0] - ax) * (center[0] - ax) + (center[1] - ay) * (center[1] - ay));
    }

    vector<Point> Delaunay(vector<Point>& points) {
        vector<Point> triAns, triTemp, edgeBuffer;

        sort(points.begin(), points.end(), [](const Point& a, const Point& b) {
            return a[0] != b[0] ? a[0] < b[0] : a[1] < b[1];
            });

        // [2] Find min and max points
        Point maxPoint = points[0], minPoint = points[0];
        for (const auto& point : points) {
            if (point[0] > maxPoint[0] || (point[0] == maxPoint[0] && point[1] > maxPoint[1])) {
                maxPoint = point;
            }
            if (point[0] < minPoint[0] || (point[0] == minPoint[0] && point[1] < minPoint[1])) {
                minPoint = point;
            }
        }

        // [3] Create supertriangle
        Point supertriangle = { minPoint[0] - 1.0f, minPoint[1] - 1.0f, maxPoint[0] + 1.0f, maxPoint[1] + 1.0f };
        triTemp.push_back(supertriangle);

        // [4] Main loop for triangulation
        for (const auto& p : points) {
            edgeBuffer.clear();

            // Remove triangles whose circumcircle contains the point
            for (auto it = triTemp.begin(); it != triTemp.end();) {
                Point center(2);
                float radius;
                circumcenter((*it), (*it) + 2, (*it) + 4, center, radius);

                if ((p[0] - center[0]) * (p[0] - center[0]) + (p[1] - center[1]) * (p[1] - center[1]) < radius * radius) {
                    edgeBuffer.push_back({ (*it)[0], (*it)[1], (*it)[2], (*it)[3] });
                    edgeBuffer.push_back({ (*it)[2], (*it)[3], (*it)[4], (*it)[5] });
                    edgeBuffer.push_back({ (*it)[4], (*it)[5], (*it)[0], (*it)[1] });
                    it = triTemp.erase(it);
                }
                else {
                    ++it;
                }
            }

            // Sort and remove duplicate edges
            sort(edgeBuffer.begin(), edgeBuffer.end());
            edgeBuffer.erase(unique(edgeBuffer.begin(), edgeBuffer.end()), edgeBuffer.end());

            // Create new triangles
            for (const auto& edge : edgeBuffer) {
                triTemp.push_back({ edge[0], edge[1], edge[2], edge[3], p[0], p[1] });
            }
        }

        // [5] Remove triangles that share vertices with supertriangle
        for (const auto& tri : triTemp) {
            if (find(supertriangle.begin(), supertriangle.end(), tri[0]) == supertriangle.end() &&
                find(supertriangle.begin(), supertriangle.end(), tri[2]) == supertriangle.end() &&
                find(supertriangle.begin(), supertriangle.end(), tri[4]) == supertriangle.end()) {
                triAns.push_back(tri);
            }
        }

        return triAns;
    }

}
#endif // COMPUTATIONAL_GEOMETRY_DELAUNAY_H
