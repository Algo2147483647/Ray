#ifndef RAY_TRACING_MANAGER_H
#define RAY_TRACING_MANAGER_H

#include <string>
#include <fstream>
#include <sstream>
#include <unordered_map>
#include <iostream>
#include "ray_tracing.h"
#include "properties/shape/shape.h"
#include "properties/shape/sphere.h"

namespace RayTracing {
    void LoadSceneFromScript(const string& filepath, ObjectTree& objTree) {
        unordered_map<string, Material*> materials;
        ifstream file(filepath);
        if (!file.is_open()) {
            cerr << "Failed to open file: " << filepath << endl;
            return;
        }

        string line;
        while (getline(file, line)) {
            // Skip empty lines and comments
            if (line.empty() || line[0] == '#') continue;

            istringstream iss(line);
            string type;
            iss >> type;

            char c;

            if (type == "Material") {
                // Parse material
                string name;
                float r, g, b, diffuse = 0, reflect = 0, refract = 0, refractivity = 1, radiate = 0;
                iss >> name >> c >> r >> c >> g >> c >> b >> c;

                auto* material = new Material({ r, g, b });
                materials[name] = material;

                string property;
                while (iss >> property) {
                    if (property.find("Diffuse=") == 0)
                        material->diffuseLoss = stof(property.substr(8));
                    else if (property.find("Reflect=") == 0)
                        material->reflectivity = stof(property.substr(8));
                    else if (property.find("Refract=") == 0)
                        material->refractivity = stof(property.substr(8));
                    else if (property.find("Refractivity=") == 0)
                        material->refractiveIndex = stof(property.substr(13));
                    else if (property.find("Radiate=") == 0)
                        material->rediate = stof(property.substr(8));
                }
            }
            else if (type == "Object") {
                // Parse object
                string shape, materialName;
                Vector3f pos1, pos2;
                float radius = 0;
                iss >> shape >> materialName;
                if (shape == "Cuboid") {
                    iss >> c >> pos1[0] >> c >> pos1[1] >> c >> pos1[2] >> c >> c
                        >> pos2[0] >> c >> pos2[1] >> c >> pos2[2] >> c;
                    objTree.Add(new Cuboid(pos1, pos2), materials[materialName]);
                }
                else if (shape == "Sphere") {
                    iss >> c >> pos1[0] >> c >> pos1[1] >> c >> pos1[2] >> c;
                    float radius = 0;
                    string property;
                    while (iss >> property) {
                        if (property.find("Radius=") == 0) {
                            radius = stof(property.substr(8));
                        }
                    }
                    objTree.Add(new Sphere(pos1, radius), materials[materialName]);
                }
            }
            else if (type == "ObjectFile") {
                // Parse external object file
                //string materialName, filePath;
                //Vector3f position;
                //float scale = 1;
                //iss >> materialName >> filePath >> c >> position[0] >> c >> position[1] >> c >> position[2] >> c >> "Scale=" >> scale;
                //objTree.Add(filePath, position, scale, materials[materialName]);
            }
        }
    file.close();
    }
}

#endif
