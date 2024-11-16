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
        Cuboid* boundbox;
        std::array<ObjectNode*, 2> children;

        ObjectNode(Object* obj, ObjectNode* leftChild = nullptr, ObjectNode* rightChild = nullptr)
            : obj(obj), children{ leftChild, rightChild } {
            if (obj != nullptr) {
                BuildComputeBoundingBox();
            }
        }

        void BuildComputeBoundingBox() {
            boundbox = &Cuboid();
            obj->shape->BuildBoundingBox(boundbox->pmax, boundbox->pmin);
        }
    };

    struct ObjectTree {
        ObjectNode* root;
        vector<Object> objects;
        vector<ObjectNode*> objectNodes;

        Object& Add(Shape* shape, Material* material = nullptr) {
            objects.emplace_back();
            Object& obj = objects.back();
            obj.shape = shape;
            obj.material = material;
            return obj;
        }

        void Build() {
            objectNodes.clear();
            for (auto& obj : objects) {
                objectNodes.push_back(&ObjectNode(&obj));
            }
            Build(objectNodes, 0, objectNodes.size() - 1, root);
        }

        void Build(vector<ObjectNode*>& objectNodes, int l, int r, ObjectNode*& node) {
            if (l == r) {
                node = objectNodes[l];
                return;
            }

            node = &ObjectNode(nullptr);
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

            node->boundbox =&Cuboid();
            node->boundbox->pmin = pmin;
            node->boundbox->pmax = pmax;

            Vector3f dim_ratios = delta.cwiseQuotient(pmax - pmin);
            int dim = (dim_ratios[0] < dim_ratios[1]) ? 1 : 0;
            dim = (dim_ratios[dim] < dim_ratios[2]) ? 2 : dim;

            sort(objectNodes.begin() + l, objectNodes.begin() + r + 1, [&dim](const ObjectNode*& a, const ObjectNode*& b) {
                if (a->boundbox->pmin[dim] != b->boundbox->pmin[dim])
                return a->boundbox->pmin[dim] < b->boundbox->pmin[dim];
            return a->boundbox->pmax[dim] < b->boundbox->pmax[dim];
                });

            Build(objectNodes, l, (l + r) / 2, node->children[0]);
            Build(objectNodes, (l + r) / 2 + 1, r, node->children[1]);
        }

        float GetIntersection(const Vector3f& raySt, const Vector3f& ray, ObjectNode*& node, Object*& obj) const {
            if (node->obj != nullptr) {
                obj = node->obj;
                return obj->shape->Intersect(raySt, ray);
            }

            if (node->boundbox->Intersect(raySt, ray) == FLT_MAX)
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