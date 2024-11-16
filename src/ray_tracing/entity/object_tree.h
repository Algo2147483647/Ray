#ifndef RAY_TRACING_OBJECT_TREE_H
#define RAY_TRACING_OBJECT_TREE_H

#include <array>
#include <vector>
#include <memory>
#include <algorithm>
#include <Eigen/Dense>
#include "object.h"
#include "../properties/shape/cuboid.h"
#include "../utils/consts.h"

using namespace Eigen;
using namespace std;

namespace RayTracing {

    struct ObjectNode {
        Object* obj;
        unique_ptr<Cuboid> boundbox;
        std::array<unique_ptr<ObjectNode>, 2> children;

        ObjectNode(Object* obj, unique_ptr<ObjectNode> leftChild = nullptr, unique_ptr<ObjectNode> rightChild = nullptr)
            : obj(obj), children{ move(leftChild), move(rightChild) } {
            if (obj != nullptr) {
                ComputeBoundingBox();
            }
        }

        void ComputeBoundingBox() {
            boundbox = make_unique<Cuboid>();
            obj->shape->boundingBox(boundbox->pmax, boundbox->pmin);
        }
    };

    struct ObjectTree {
        unique_ptr<ObjectNode> root;
        vector<Object> objects;
        vector<unique_ptr<ObjectNode>> objectNodes;

        Object& Add(unique_ptr<Shape> shape, unique_ptr<Material> material = nullptr) {
            objects.emplace_back();
            Object& obj = objects.back();
            obj.shape = move(shape);
            obj.material = move(material);
            return obj;
        }

        void build() {
            objectNodes.clear();
            for (auto& obj : objects) {
                objectNodes.push_back(make_unique<ObjectNode>(&obj));
            }
            build(objectNodes, 0, static_cast<int>(objectNodes.size()) - 1, root);
        }

        void build(vector<unique_ptr<ObjectNode>>& objectNodes, int l, int r, unique_ptr<ObjectNode>& node) {
            if (l == r) {
                node = move(objectNodes[l]);
                return;
            }

            node = make_unique<ObjectNode>(nullptr);
            Vector3f pmin = objectNodes[l]->boundbox->pmin;
            Vector3f pmax = objectNodes[l]->boundbox->pmax;
            Vector3f delta = Vector3f::Zero();

            for (int i = l + 1; i <= r; ++i) {
                for (int j = 0; j < 3; ++j) {
                    pmin[j] = min(pmin[j], objectNodes[i]->boundbox->pmin[j]);
                    pmax[j] = max(pmax[j], objectNodes[i]->boundbox->pmax[j]);
                    delta[j] = max(delta[j], objectNodes[i]->boundbox->pmax[j] - objectNodes[i]->boundbox->pmin[j]);
                }
            }

            node->boundbox = make_unique<Cuboid>();
            node->boundbox->pmin = pmin;
            node->boundbox->pmax = pmax;

            Vector3f dim_ratios = delta.cwiseQuotient(pmax - pmin);
            int dim = (dim_ratios[0] < dim_ratios[1]) ? 1 : 0;
            dim = (dim_ratios[dim] < dim_ratios[2]) ? 2 : dim;

            sort(objectNodes.begin() + l, objectNodes.begin() + r + 1, [&dim](const unique_ptr<ObjectNode>& a, const unique_ptr<ObjectNode>& b) {
                if (a->boundbox->pmin[dim] != b->boundbox->pmin[dim])
                return a->boundbox->pmin[dim] < b->boundbox->pmin[dim];
            return a->boundbox->pmax[dim] < b->boundbox->pmax[dim];
                });

            build(objectNodes, l, (l + r) / 2, node->children[0]);
            build(objectNodes, (l + r) / 2 + 1, r, node->children[1]);
        }

        float GetIntersection(const Vector3f& raySt, const Vector3f& ray, unique_ptr<ObjectNode>& node, Object*& obj) const {
            if (node->obj != nullptr) {
                obj = node->obj;
                return obj->shape->intersect(raySt, ray);
            }

            if (node->boundbox->intersect(raySt, ray) == FLT_MAX)
                return FLT_MAX;

            Object* ob_1, * ob_2;
            float dis_1 = GetIntersection(raySt, ray, node->children[0], ob_1);
            float dis_2 = GetIntersection(raySt, ray, node->children[1], ob_2);
            dis_1 = dis_1 > EPS ? dis_1 : FLT_MAX;
            dis_2 = dis_2 > EPS ? dis_2 : FLT_MAX;

            obj = dis_1 < dis_2 ? ob_1 : ob_2;
            return min(dis_1, dis_2);
        }
    };
}
#endif