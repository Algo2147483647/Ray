#include <vector>
#include <utility>
#include <corecrt_math_defines.h>

using namespace std;

namespace Modeling {
    typedef vector<double> Point;

	inline vector<Point>& RegularPolygon(vector<Point>& res, int n, double a0 = 0) {
        double a = 2 * M_PI / n;

        for (int i = 0; i < n; i++)
            res.push_back({ cos(a0 + a * i), sin(a0 + a * i) });

        return res;
    }

	inline vector<Point>& getSphereFibonacciPoint(vector<Point>& res, int n) {
		double angleIncrement = M_PI * 2 * (1 + sqrt(5)) / 2;	// �ƽ�ָ��

		for (int i = 0; i < n; i++) {
			double t = (double)i / n;
			double inclination = acos(1 - 2 * t);
			double azimuth = angleIncrement * i;

			res.push_back({
				sin(inclination)* cos(azimuth), 
				sin(inclination)* sin(azimuth),
				cos(inclination)
			});
		}
		return res;
	}

}
