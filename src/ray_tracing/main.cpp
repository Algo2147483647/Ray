#include <chrono>
#include <iostream>
#include "ray_tracing.h"
#include "ray_tracing_manager.h"
#include "../image/ppm.h"

using namespace RayTracing;


int main(int argc, char* argv[]) {
    // 检查命令行参数
    if (argc < 2) {
        std::cerr << "Usage: " << argv[0] << " <script_file_path>" << std::endl;
        return 1;
    }

    // 加载脚本文件
    std::string scriptPath = argv[1];
    ObjectTree objTree;

    try {
        LoadSceneFromScript(scriptPath, objTree);
    }
    catch (const std::exception& e) {
        std::cerr << "Error loading script: " << e.what() << std::endl;
        return 1;
    }

    // 设置摄像机参数
    vector<MatrixXf> img(3, MatrixXf(800, 800));

    Camera camera;
    camera.position = Vector3f(600, 1100, 600);
    Vector3f lookAt = Vector3f(400, -100, -100);
    camera.SetLookAt(lookAt);
    

    // 开始渲染
    auto start = std::chrono::high_resolution_clock::now();
    RayTracing::TraceRay(camera, objTree.Build(), img, 0, 100);
    auto stop = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(stop - start);
    std::cout << "Time taken by traceRay: " << duration.count() << " milliseconds" << std::endl;

    // 保存渲染结果
    ImageRGB imgout(800, 800);
    for (int i = 0; i < 800; i++) {
        for (int j = 0; j < 800; j++) {
            imgout(i, j) = RGB(
                std::min((int)(img[0](i, j) * 255), 255),
                std::min((int)(img[1](i, j) * 255), 255),
                std::min((int)(img[2](i, j) * 255), 255));
        }
    }

    // 输出文件路径
    std::string outputPath = "output.ppm";
    PPMWrite(outputPath, imgout);
    std::cout << "Rendering completed. Output saved to " << outputPath << std::endl;

    return 0;
}