#ifndef RAY_TRACING_OBJECT_H
#define RAY_TRACING_OBJECT_H

#include <vector>
#include <memory>
#include <algorithm>
#include <Eigen/Dense>
#include "Material.h"
#include "Shape.h"
#include "GraphicsIO.h"

using namespace Eigen;
using namespace std;

/*#############################################################################
*
*					对象/对象树
*
##############################################################################*/
namespace RayTracing {

	class Object {
	public:
		Shape* shape;
		Material* material;
		//Texture* texture;
	};

	class ObjectNode {
	public:
		Object* obj;
		Cuboid* boundbox;
		ObjectNode* kid[2];

		ObjectNode(Object* obj, ObjectNode* leftKid = nullptr, ObjectNode* rightKid = nullptr)
			: obj(obj) {
			if(obj != nullptr)
				computeBoundingBox();
			kid[0] = leftKid;
			kid[1] = rightKid;
		}

		void computeBoundingBox() {
			boundbox = new Cuboid;
			obj->shape->boundingBox(boundbox->pmax, boundbox->pmin);
		}
	};

	class ObjectTree {
	public:
		ObjectNode* root = nullptr;
		vector<Object> ObjectSet;
		vector<ObjectNode> ObjectNodeSet;

		inline Object& add(Shape* shape, Material* material = nullptr) {
			Object obj;

			obj.shape = shape;
			obj.material = material;

			ObjectSet.push_back(obj);
			return ObjectSet.back();
		}

		inline Object& add(const char* file, Vector3f center, double size, Material* material) {
			Mat<float> p0, p1, p2, p3;
			Vector3f p4, p5, p6;
			vector<short> a;
			Graphics::stlRead(file, p0, p1, p2, p3, a);

			for (int i = 0; i < p0.cols(); i++) {
				p4 = Vector3f(p1(0, i), p1(1, i), p1(2, i)) * size + center;
				p5 = Vector3f(p2(0, i), p2(1, i), p2(2, i)) * size + center;
				p6 = Vector3f(p3(0, i), p3(1, i), p3(2, i)) * size + center;
				add(new Triangle(p4, p5, p6), material);
			}
		}

		/*--------------------------------[ 建树 ]--------------------------------*/
		inline void build() {
			ObjectNodeSet.clear();
			for (int i = 0; i < ObjectSet.size(); i++) {
				ObjectNodeSet.push_back({ &ObjectSet[i] });
			}

			build(ObjectNodeSet, 0, ObjectNodeSet.size() - 1, root);
		}

		inline void build(vector<ObjectNode>& ObjectNodeSet, const int l, const int r, ObjectNode*& node) {
			if (l == r) {
				node = &ObjectNodeSet[l];
				return;
			}
			
			node = new ObjectNode(nullptr);
			Vector3f pmin = ObjectNodeSet[l].boundbox->pmin;
			Vector3f pmax = ObjectNodeSet[l].boundbox->pmax;
			Vector3f delta = Vector3f::Zero();

			for (int i = l + 1; i <= r; ++i) {
				for (int j = 0; j < 3; ++j) {
					pmin[j] = min(pmin[j], ObjectNodeSet[i].boundbox->pmin[j]);
					pmax[j] = max(pmax[j], ObjectNodeSet[i].boundbox->pmax[j]);
					delta[j] = max(delta[j], ObjectNodeSet[i].boundbox->pmax[j] - ObjectNodeSet[i].boundbox->pmin[j]);
				}
			}

			node->boundbox = new Cuboid;
			node->boundbox->pmin = pmin;
			node->boundbox->pmax = pmax;

			Vector3f dim_ratios = delta.cwiseQuotient(pmax - pmin);
			int dim = 0;
			dim = (dim_ratios[dim] < dim_ratios[1]) ? dim : 1;
			dim = (dim_ratios[dim] < dim_ratios[2]) ? dim : 2;

			sort(ObjectNodeSet.begin() + l, ObjectNodeSet.begin() + r + 1, [&dim](const ObjectNode& a, const ObjectNode& b) {
				if (a.boundbox->pmin[dim] != b.boundbox->pmin[dim])
					return a.boundbox->pmin[dim] < b.boundbox->pmin[dim];
				return a.boundbox->pmax[dim] < b.boundbox->pmax[dim];
			});

			build(ObjectNodeSet, l, (l + r) / 2, node->kid[0]);
			build(ObjectNodeSet, (l + r) / 2 + 1, r, node->kid[1]);
		}

		/*--------------------------------[ 求交 ]--------------------------------*/
		inline float seekIntersection(const Vector3f& raySt, const Vector3f& ray, Object*& obj) const {
			return seekIntersection(raySt, ray, root, obj);
		}

		inline float seekIntersection(const Vector3f& raySt, const Vector3f& ray, const ObjectNode* node, Object*& obj) const {
			if (node->obj != nullptr) {
				obj = node->obj;
				return obj->shape->intersect(raySt, ray);
			}

			if (node->boundbox->intersect(raySt, ray) == FLT_MAX)
				return FLT_MAX;

			Object* ob_1, * ob_2;
			float dis_1 = seekIntersection(raySt, ray, node->kid[0], ob_1);
			float dis_2 = seekIntersection(raySt, ray, node->kid[1], ob_2);
			dis_1 = dis_1 > EPS ? dis_1 : FLT_MAX;
			dis_2 = dis_2 > EPS ? dis_2 : FLT_MAX;

			obj = dis_1 < dis_2 ? ob_1 : ob_2;
			return min(dis_1, dis_2);
		}

	};
}
#endif