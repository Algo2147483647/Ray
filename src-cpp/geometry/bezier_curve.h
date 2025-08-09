#ifndef GEOMETRY_BEZIERCURVE_H
#define GEOMETRY_BEZIERCURVE_H

#include <vector>

using namespace std;

template<typename T>
inline vector<T>& BezierCurve(vector<vector<T>>& points, T t, vector<T>& res) {
	int n = points.size();
	vector<vector<T>> c (points);

	for (int i = 1; i < n; i++)
		for (int k = 0; k < n - i; k++)
			for (int d = 0; d < c[0].size(); d++)
				c[k][d] = c[k][d] * (1 - t) + c[k + 1][d] * t;

	return res = c[0];
}

template<typename T>
inline vector<vector<T>>& BezierCurve(vector<vector<T>>& points, int n, vector<vector<T>>& res) {
	res.resize(n + 1);
	for (int i = 0; i <= n; i++) {
		BezierCurve(points, i / (T)n, res[i]);
	}
	return res;
}

#endif