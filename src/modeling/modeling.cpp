#include "Modeling.h"


void Modeling::writeModel(const char* fileName) {
	char head[80] = { 0 };
	Mat<float> p[3], fv;
	Mat<float> t;

	p[0].resize(3, Object.size());
	p[1].resize(3, Object.size());
	p[2].resize(3, Object.size());

	t.resize(3, Object.size()).fill(1);
	t.normalize();

	fv.resize(3, Object.size());
	for (int i = 0; i < t.size(); i++)
		fv(i) = t(i);

	Mat<short> attr(Object.size());

	for (int tri = 0; tri < Object.size(); tri++)
		for (int poi = 0; poi < 3; poi++)
			for (int dim = 0; dim < 3; dim++)
				p[poi](dim, tri) = Object[tri][poi * 3 + dim];

	Graphics::stlWrite(fileName, head, fv, p[0], p[1], p[2], attr);
}


void Modeling::Rotator(Point& center, Vector3f& axis, vector<Point>& f, int pointNum, float st, float ed) {
	float dAngle = (ed - st) / pointNum;
	Point p1, p2, p3, p4;
	Mat<float> rotateMat_0(3, 3), rotateMat(3, 3), preRotateMat(3, 3), firstRotateMat(3, 3);
	Vector3f direction, delta, direction_2;

	axis.normalize();

	// calculate the first rotate matrix
	float
		x = axis[0],
		y = axis[1],
		z = axis[2],
		sign = x * y > 0 ? -1 : 1,
		e = sqrt(1 - z * z),
		b = -x * z / e,
		d = -y * z / e,
		a = sign * abs(y / e),
		c = abs(x / e);

	if (e < 1e-4)
		E(rotateMat_0);
	else
		rotateMat_0 <<
		a, b, x,
		c, d, y,
		0, e, z;

	for (int i = 0; i <= pointNum; i++) {
		float angle = st + dAngle * i;
		delta = { cos(angle), sin(angle), 0 };
		direction = (rotateMat_0 * delta).normalized();
		normalize(cross(direction_2, axis, direction));

		// calculate rotate matrix
		rotateMat <<
			direction[0], axis[0], direction_2[0],
			direction[1], axis[1], direction_2[1],
			direction[2], axis[2], direction_2[2];

		// generate
		if (i != 0) {
			for (int j = 1; j < f.size(); j++) {
				p3 = p1 = { f[j - 1][0], f[j - 1][1], 0 };
				p4 = p2 = { f[j][0],     f[j][1],     0 };

				p1 = center + rotateMat * p1;
				p2 = center + rotateMat * p2;
				p3 = center + preRotateMat * p3;
				p4 = center + preRotateMat * p4;
				Quadrangle(p1, p2, p3, p4);
			}
		}
		preRotateMat = rotateMat;
		if (i == 0)
			firstRotateMat = rotateMat;
	}
}


void Modeling::Rotator(Point& center, Point& axis, vector<Point>& f, int pointNum, float st, float ed, int isClosed) {
	float dAngle = (ed - st) / pointNum;
	Point p1, p2, p3, p4;
	Mat<float> rotateMat_0(3, 3), rotateMat(3, 3), preRotateMat(3, 3), firstRotateMat(3, 3);
	Vector3f direction, delta, direction_2;

	axis.normalize();

	// calculate the first rotate matrix
	float
		x = axis[0],
		y = axis[1],
		z = axis[2],
		sign = x * y > 0 ? -1 : 1,
		e = sqrt(1 - z * z),
		b = -x * z / e,
		d = -y * z / e,
		a = sign * abs(y / e),
		c = abs(x / e);

	if (e < 1e-4)
		E(rotateMat_0);
	else 
		rotateMat_0 <<
			a, b, x,
			c, d, y,
			0, e, z
		;

	for (int i = 0; i <= pointNum; i++) {
		float angle = st + dAngle * i;
		delta = { cos(angle), sin(angle), 0 };
		direction = (rotateMat_0 * delta).normalized();
		normalize(cross(direction_2, axis, direction));

		// calculate rotate matrix
		rotateMat <<
			direction[0], axis[0], direction_2[0],
			direction[1], axis[1], direction_2[1],
			direction[2], axis[2], direction_2[2];

		// generate
		if (i != 0) {
			for (int j = 1; j < f.size(); j++) {
				p3 = p1 = { f[j - 1][0], f[j - 1][1], 0 };
				p4 = p2 = { f[j][0],     f[j][1],     0 };

				p1 = center + rotateMat * p1;
				p2 = center + rotateMat * p2;
				p3 = center + preRotateMat * p3;
				p4 = center + preRotateMat * p4;
				Triangle(p1, p2, p3);
				Triangle(p4, p3, p2);
			}
		}
		preRotateMat = rotateMat;
		if (i == 0)
			firstRotateMat = rotateMat;
	}

	// closed
	if (isClosed) {
		Point p1, p2, p3;
		vector<vector<float>> tris;

		Graphics::earClippingTriangulation(f, tris);

		int n = tris.size();
		
		for (int i = 0; i < n; i++) {
			p1 = rotateMat * Vector3f(tris[i][0], tris[i][1], tris[i][2]);
			p2 = rotateMat * Vector3f(tris[i][3], tris[i][4], tris[i][5]);
			p3 = rotateMat * Vector3f(tris[i][6], tris[i][7], tris[i][8]);

			Object.push_back({
				p1[0] + center[0], p1[1] + center[1], p1[2] + center[2],
				p2[0] + center[0], p2[1] + center[1], p2[2] + center[2],
				p3[0] + center[0], p3[1] + center[1], p3[2] + center[2]
			});

			p1 = firstRotateMat * Vector3f(tris[i][0], tris[i][1], tris[i][2]);
			p2 = firstRotateMat * Vector3f(tris[i][3], tris[i][4], tris[i][5]);
			p3 = firstRotateMat * Vector3f(tris[i][6], tris[i][7], tris[i][8]);

			Object.push_back({
				p1[0] + center[0], p1[1] + center[1], p1[2] + center[2],
				p2[0] + center[0], p2[1] + center[1], p2[2] + center[2],
				p3[0] + center[0], p3[1] + center[1], p3[2] + center[2]
			});
		}
	}
}


void Modeling::Translator(Point& st, Point& ed, vector<Point>& f, int isClosed) {
	// calculate rotate matrix
	Mat<float> rotateMat(3, 3);
	Vector3f direction = (ed - st).normalized();

	float
		x = direction[0],
		y = direction[1],
		z = direction[2],
		sign = x * y > 0 ? -1 : 1,
		e = sqrt(1 - z * z),
		b = -x * z / e,
		d = -y * z / e,
		a = sign * abs(y / e),
		c = abs(x / e);

	if (e < 1e-4)
		E(rotateMat);
	else
		rotateMat <<
			a, b, x,
			c, d, y,
			0, e, z;

	// generate
	Point stPoint, edPoint, preStPoint, preEdPoint;

	int fn = f.size();
	Point pt;

	for (int i = 0; i < fn; i++) {
		pt = rotateMat * Vector3f(f[i][0], f[i][1], 0);
		stPoint = st + pt;
		edPoint = ed + pt;

		if (i != 0) 
			Quadrangle(stPoint, preStPoint, edPoint, preEdPoint);
		
		preStPoint = stPoint;
		preEdPoint = edPoint;
	}

	// closed
	if (isClosed) {
		Point p1, p2, p3;
		vector<vector<float>> tris;

		Graphics::earClippingTriangulation(f, tris);

		int n = tris.size();

		for (int i = 0; i < n; i++) {
			p1 = rotateMat * Vector3f(tris[i][0], tris[i][1], tris[i][2]);
			p2 = rotateMat * Vector3f(tris[i][3], tris[i][4], tris[i][5]);
			p3 = rotateMat * Vector3f(tris[i][6], tris[i][7], tris[i][8]);

			Object.push_back({
				p1[0] + st[0], p1[1] + st[1], p1[2] + st[2],
				p2[0] + st[0], p2[1] + st[1], p2[2] + st[2],
				p3[0] + st[0], p3[1] + st[1], p3[2] + st[2]
			});
			Object.push_back({
				p1[0] + ed[0], p1[1] + ed[1], p1[2] + ed[2],
				p2[0] + ed[0], p2[1] + ed[1], p2[2] + ed[2],
				p3[0] + ed[0], p3[1] + ed[1], p3[2] + ed[2]
			});
		}
	}
}


void Modeling::Translator(vector<Point>& path, vector<Point>& f, int isClosed) {
	int n = path.size();
	Point p1 = path[0], p2;

	for (int i = 1; i < n; i++) {
		p2 = path[i];

		if (isClosed && (i == 1 || i == n - 1))
			Translator(p1, p2, f, true);
		else
			Translator(p1, p2, f, false);

		p1 = p2;
	}
}


void Modeling::Rotator_Translator(
	Point& center, Point& axis, vector<Point>& f,
	vector<float>& direction_, float length,
	int pointNum, float st, float ed
) {
	float dAngle = (ed - st) / pointNum;
	Point p, p1, p2, p3, p4;
	Mat<float> rotateMat_0(3, 3), rotateMat(3, 3), preRotateMat(3, 3);
	Vector3f direction, delta, direction_2;

	axis.normalize();

	// calculate the first rotate matrix
	float
		x = axis[0],
		y = axis[1],
		z = axis[2],
		sign = x * y > 0 ? -1 : 1,
		e = sqrt(1 - z * z),
		b = -x * z / e,
		d = -y * z / e,
		a = sign * abs(y / e),
		c = abs(x / e);

	if (e < 1e-4)
		E(rotateMat_0);
	else
		rotateMat_0 <<
			a, b, x,
			c, d, y,
			0, e, z;

	for (int i = 0; i <= pointNum; i++) {
		float angle = st + dAngle * i;
		delta = { cos(angle), sin(angle), 0 };
		direction = (rotateMat_0 * delta).normalized();
		normalize(cross(direction_2, axis, direction));

		// calculate rotate matrix
		rotateMat <<
			direction[0], axis[0], direction_2[0],
			direction[1], axis[1], direction_2[1],
			direction[2], axis[2], direction_2[2];

		// generate
		if (i != 0) {
			for (int j = 1; j < f.size(); j++) {
				p3 = p1 = { f[j - 1][0], f[j - 1][1], 0 };
				p4 = p2 = { f[j][0],     f[j][1],     0 };

				mul(p1, rotateMat, p1);
				mul(p2, rotateMat, p2);
				mul(p3, preRotateMat, p3);
				mul(p4, preRotateMat, p4);

				add(p1, p1, center);
				add(p2, p2, center);
				add(p3, p3, center);
				add(p4, p4, center);

				add(p1, p1, mul(p, i / (float)pointNum * length, direction_));
				add(p2, p2, mul(p, i / (float)pointNum * length, direction_));
				add(p3, p3, mul(p, (i - 1) / (float)pointNum * length, direction_));
				add(p4, p4, mul(p, (i - 1) / (float)pointNum * length, direction_));

				Triangle(p1, p2, p3);
				Triangle(p4, p3, p2);
			}
		}
		preRotateMat = rotateMat;
	}
}


void Modeling::Triangle(Point& p1, Point& p2, Point& p3) {
	triangle tri(9);

	for (int i = 0; i < 3; i++) {
		tri[i] = p1[i];
		tri[i + 3] = p2[i];
		tri[i + 6] = p3[i];
	}

	Object.push_back(tri);
}


void Modeling::Rectangle(Point& c, float X, float Y) {
	Point p1, p2, p3;
	Triangle(
		p1 = { c[0] + X / 2, c[1] + Y / 2, c[2] },
		p2 = { c[0] + X / 2, c[1] - Y / 2, c[2] },
		p3 = { c[0] - X / 2, c[1] + Y / 2, c[2] }
	);
	Triangle(
		p1 = { c[0] - X / 2, c[1] - Y / 2, c[2] },
		p2 = { c[0] + X / 2, c[1] - Y / 2, c[2] },
		p3 = { c[0] - X / 2, c[1] + Y / 2, c[2] }
	);
}


void Modeling::Quadrangle(Point& p1, Point& p2, Point& p3, Point& p4) {
	Triangle(p1, p2, p3);
	Triangle(p1, p3, p4);
}


void Modeling::ConvexPolygon(vector<Point>& p) {
	int n = p.size();

	for (int k = 1; k <= (n + 2) / 3; k++)
		for (int i = 0; i <= n - 2 * k; i += 2 * k)
			Triangle(p[i], p[i + k], p[(i + 2 * k) % n]);
}

void Modeling::Polygon(Point& c, vector<Point>& p) {
	vector<vector<float>> tris;

	Graphics::earClippingTriangulation(p, tris);
	addTriangleSet(c, tris);
}


void Modeling::Circle(Point& center, float r, int pointNum, float angleSt, float angleEd) {
	float dAngle = (angleEd - angleSt) / pointNum;
	Point ps, pe;

	for (int i = 0; i < pointNum; i++) {
		float theta = i * dAngle;
		ps = Point(
			r * cos(theta + angleSt),
			r * sin(theta + angleSt),
			0
		) + center;
		pe = Point(
			r * cos(theta + angleSt + dAngle),
			r * sin(theta + angleSt + dAngle),
			0
		) + center;
		Triangle(ps, pe, center);
	}
}


void Modeling::Surface(Mat<float>& z, float xs, float xe, float ys, float ye, Point* direct) {
	Point p, pl, pu, plu; 

	float 
		dx = (xe - xs) / z.rows(),
		dy = (ye - ys) / z.cols();

	for (int y = 0; y < z.cols(); y++) {
		for (int x = 0; x < z.rows(); x++) {
			if (z(x, y) == HUGE_VAL) 
				continue;

			p = { 
				xs + x * dx, 
				ys + y * dy, 
				z(x, y)
			};

			if (x == 0 || y == 0 ||
				z(x - 1, y) == HUGE_VAL ||  
				z(x, y - 1) == HUGE_VAL ||  
				z(x - 1, y - 1) == HUGE_VAL
			) continue;

			pl = { 
				xs + (x - 1) * dx, 
				ys + y * dy,
				z(x - 1, y) 
			};
			pu = {
				xs + x * dx,
				ys + (y - 1) * dy, 
				z(x, y - 1) 
			};
			plu = { 
				xs + (x - 1) * dx,	
				ys + (y - 1) * dy,
				z(x - 1, y - 1) 
			};

			Triangle(p, pl, pu);
			Triangle(plu, pu, pl);
		}
	}
}


void Modeling::Tetrahedron(Point& p1, Point& p2, Point& p3, Point& p4) {
	Triangle(p1, p2, p3);
	Triangle(p2, p3, p4);
	Triangle(p3, p4, p1);
	Triangle(p4, p1, p2);
}


void Modeling::Cuboid(Point& pMin, Point& pMax) {
	Point pMinTmp[3], pMaxTmp[3];
	for (int i = 0; i < 3; i++) {
		pMinTmp[i] = pMin; pMinTmp[i][i] = pMax[i];
		pMaxTmp[i] = pMax; pMaxTmp[i][i] = pMin[i];
	}

	for (int i = 0; i < 3; i++) {
		Quadrangle(pMin, pMinTmp[i], pMaxTmp[(i + 2) % 3], pMinTmp[(i + 1) % 3]);
		Quadrangle(pMax, pMaxTmp[i], pMinTmp[(i + 2) % 3], pMaxTmp[(i + 1) % 3]);
	}
}


void Modeling::Cuboid(Point& center, float X, float Y, float Z) {
	Point delta = { X / 2, Y / 2, Z / 2 };
	Point pMax = center + delta;
	Point pMin = center - delta;
	Cuboid(pMin, pMax);
}


void Modeling::Cuboid(Point& center, Vector3f& direction, float L, float W, float H) {
	vector<Point> f;
	Point p, st, ed;
	f = {
		{-W / 2,-H / 2},
		{+W / 2,-H / 2},
		{+W / 2,+H / 2},
		{-W / 2,+H / 2},
		{-W / 2,-H / 2},
	};

	direction.normalize();

	ed = center + direction * ( L / 2.0);
	st = center + direction * (-L / 2.0);
	Translator(st, ed, f);
}


void Modeling::Frustum(Point& st, Point& ed, float Rst, float Red, int pointNum) {
	;
}


void Modeling::Sphere(Point& center, float r, int pointNum) {
	Vector3f st, ed;
	vector<int> N;
	vector<vector<float>> triangleSet;
	float more = r / pointNum * 3;

	Graphics::MarchingCubes([&](float x, float y, float z) {
		return r * r - (x * x + y * y + z * z); 
		},
		st = {-r - more,-r - more,-r - more },
		ed = { r + more, r + more, r + more },
		N  = { pointNum, pointNum, pointNum },
		triangleSet
	);

	addTriangleSet(center, triangleSet);
}


void Modeling::Sphere(Point& center, float r, int ThetaNum, int PhiNum, 
	float thetaSt, float thetaEd, 
	float phiSt, float phiEd
) {
	Point point, pointU, pointL, pointUL;
	float
		dTheta = (thetaEd - thetaSt) / ThetaNum,
		dPhi   = (phiEd - phiSt)     / PhiNum;

	for (int i = 1; i <= ThetaNum; i++) {
		float theta = thetaSt + i * dTheta;

		for (int j = 1; j <= PhiNum; j++) {
			float phi = phiSt + j * dPhi;

			point = {
				r * cos(phi) * cos(theta) + center[0],
				r * cos(phi) * sin(theta) + center[1],
				r * sin(phi) + center[2]
			};
			pointU = {
				r * cos(phi - dPhi) * cos(theta) + center[0],
				r * cos(phi - dPhi) * sin(theta) + center[1],
				r * sin(phi - dPhi) + center[2]
			};
			pointL = {
				r * cos(phi) * cos(theta - dTheta) + center[0],
				r * cos(phi) * sin(theta - dTheta) + center[1],
				r * sin(phi) + center[2]
			};
			pointUL = {
				r * cos(phi - dPhi) * cos(theta - dTheta) + center[0],
				r * cos(phi - dPhi) * sin(theta - dTheta) + center[1],
				r * sin(phi - dPhi) + center[2]
			};

			Triangle(point,  pointU, pointL);
			Triangle(pointL, pointU, pointUL);
		}
	}
}


void Modeling::addTriangleSet(Point& center, vector<triangle>& tris) {
	int n = tris.size();

	for (int i = 0; i < n; i++) {
		Object.push_back({
			tris[i][0] + center[0], tris[i][1] + center[1], tris[i][2] + center[2],
			tris[i][3] + center[0], tris[i][4] + center[1], tris[i][5] + center[2],
			tris[i][6] + center[0], tris[i][7] + center[1], tris[i][8] + center[2],
		});
	}
}


void Modeling::Array(int count, float dx, float dy, float dz) {
	int n = Object.size();
	Point p1, p2, p3, delta;
	delta = { dx, dy, dz };

	for (int k = 1; k < count; k++) {
		for (int tri = 0; tri < n; tri++) {
			for (int dim = 0; dim < 3; dim++) {
				p1[dim] = Object[tri][dim];
				p2[dim] = Object[tri][3 + dim];
				p3[dim] = Object[tri][6 + dim];

				p1[dim] += k * delta[dim];
				p2[dim] += k * delta[dim];
				p3[dim] += k * delta[dim];
			}
			Triangle(p1, p2, p3);
		}
	}
}