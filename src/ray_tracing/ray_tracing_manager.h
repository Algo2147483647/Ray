#ifndef RAY_TRACING_MANAGER_H
#define RAY_TRACING_MANAGER_H

#include <string>
#include <fstream>
#include <sstream>
#include <unordered_map>
#include <iostream>
#include "ray_tracing.h"

namespace RayTracing {
    void loadSceneFromScript(const std::string& filepath, ObjectTree& objTree) {
        std::unordered_map<std::string, Material*> materials;
        std::ifstream file(filepath);
        if (!file.is_open()) {
            std::cerr << "Failed to open file: " << filepath << std::endl;
            return;
        }

        std::string line;
        while (std::getline(file, line)) {
            // Skip empty lines and comments
            if (line.empty() || line[0] == '#') continue;

            std::istringstream iss(line);
            std::string type;
            iss >> type;

            if (type == "Material") {
                // Parse material
                std::string name;
                float r, g, b, diffuse = 0, reflect = 0, refract = 0, refractivity = 1, radiate = 0;
                char c;
                iss >> name >> c >> r >> c >> g >> c >> b >> c;
                std::string property;
                while (iss >> property) {
                    if (property.find("Diffuse=") == 0)
                        diffuse = std::stof(property.substr(8));
                    else if (property.find("Reflect=") == 0)
                        reflect = std::stof(property.substr(8));
                    else if (property.find("Refract=") == 0)
                        refract = std::stof(property.substr(8));
                    else if (property.find("Refractivity=") == 0)
                        refractivity = std::stof(property.substr(13));
                    else if (property.find("Radiate=") == 0)
                        radiate = std::stof(property.substr(8));
                }
                auto* material = new Material({ r, g, b });
                material->diffuseReflectProbability = diffuse;
                material->reflectProbability = reflect;
                material->refractProbability = refract;
                material->refractivity[0] = refractivity;
                material->rediate = radiate;
                materials[name] = material;

            }
            else if (type == "Object") {
                // Parse object
                std::string shape, materialName;
                Vector3f pos1, pos2;
                float radius = 0;
                iss >> shape >> materialName;
                if (shape == "Cuboid") {
                    iss >> c >> pos1[0] >> c >> pos1[1] >> c >> pos1[2] >> c >> c
                        >> pos2[0] >> c >> pos2[1] >> c >> pos2[2] >> c;
                    objTree.add(new Cuboid(pos1, pos2), materials[materialName]);
                }
                else if (shape == "Sphere") {
                    iss >> c >> pos1[0] >> c >> pos1[1] >> c >> pos1[2] >> c >> "Radius=" >> radius;
                    objTree.add(new Sphere(pos1, radius), materials[materialName]);
                }

            }
            else if (type == "ObjectFile") {
                // Parse external object file
                std::string materialName, filePath;
                Vector3f position;
                float scale = 1;
                iss >> materialName >> filePath >> c >> position[0] >> c >> position[1] >> c >> position[2] >> c >> "Scale=" >> scale;
                objTree.add(filePath, position, scale, materials[materialName]);
            }
        }
    file.close();
    }

    inline void RayTracingTest() {
        ObjectTree objTree;
        Camera camera(1000, 1000, Vector3f(600, 1100, 600), Vector3f(400, -100, -100));
        vector<MatrixXf> img(3, MatrixXf(800, 800));

        // Load scene from script
        loadSceneFromScript("C:/path/to/scene.txt", objTree);
        objTree.build();
        RayTracing::debug(camera, objTree);

        auto start = std::chrono::high_resolution_clock::now();
        RayTracing::traceRay(camera, objTree, img, 0, 1);
        auto stop = std::chrono::high_resolution_clock::now();
        auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(stop - start);
        std::cout << "Time taken by traceRay: " << duration.count() << " milliseconds" << std::endl;

        Image imgout(800, 800);
        for (int i = 0; i < 800; i++)
            for (int j = 0; j < 800; j++)
                imgout(i, j) = RGB(
                    min((int)(img[0](i, j) * 255), 255),
                    min((int)(img[1](i, j) * 255), 255),
                    min((int)(img[2](i, j) * 255), 255)).to_ARGB();

        Graphics::ppmWrite("C:/path/to/output.ppm", imgout);
    }
}

#endif
