#include <chrono>
#include <iostream>
#include "ray_tracing.h"

using namespace RayTracing;


int main(int argc, char* argv[]) {
    // 检查命令行参数
    if (argc < 2) {
        std::cerr << "Usage: " << argv[0] << " <script_file_path>" << std::endl;
        return 1;
    }

    std::string scriptPath = argv[1];
    ObjectTree objTree;

    // 加载脚本文件
    try {
        loadSceneFromScript(scriptPath, objTree);
    }
    catch (const std::exception& e) {
        std::cerr << "Error loading script: " << e.what() << std::endl;
        return 1;
    }

    // 设置摄像机参数
    Camera camera(1000, 1000, Vector3f(600, 1100, 600), Vector3f(400, -100, -100));
    std::vector<MatrixXf> img(3, MatrixXf(800, 800));

    // 构建场景
    objTree.Build();

    // 开始渲染
    auto start = std::chrono::high_resolution_clock::now();
    RayTracing::traceRay(camera, objTree, img, 0, 1);
    auto stop = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(stop - start);
    std::cout << "Time taken by traceRay: " << duration.count() << " milliseconds" << std::endl;

    // 保存渲染结果
    Image imgout(800, 800);
    for (int i = 0; i < 800; i++) {
        for (int j = 0; j < 800; j++) {
            imgout(i, j) = RGB(
                std::min((int)(img[0](i, j) * 255), 255),
                std::min((int)(img[1](i, j) * 255), 255),
                std::min((int)(img[2](i, j) * 255), 255)).to_ARGB();
        }
    }

    // 输出文件路径
    std::string outputPath = "output.ppm";
    Graphics::ppmWrite(outputPath, imgout);
    std::cout << "Rendering completed. Output saved to " << outputPath << std::endl;

    return 0;
}