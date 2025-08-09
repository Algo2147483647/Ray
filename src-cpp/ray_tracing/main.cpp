#include <chrono>
#include <iostream>
#include <fstream>
#include <string>
#include "ray_tracing.h"
#include "ray_tracing_manager.h"
#include "../image/ppm.h"

using namespace RayTracing;
using namespace std;

class Handler {
private:
    bool has_error;
    string error_msg;
    string scriptPath;

    // 渲染过程数据
    ObjectTree objTree;
    Camera camera;
    vector<MatrixXf> img;
    ImageRGB imgout;

public:
    Handler() : has_error(false) {}

    Handler& PreCheck() {
        if (has_error) return *this;
        // 检查脚本文件是否存在
        ifstream file(scriptPath);
        if (!file.good()) {
            has_error = true;
            error_msg = "Script file not found: " + scriptPath;
        }
        return *this;
    }

    Handler& LoadScript() {
        if (has_error) return *this;
        try {
            LoadSceneFromScript(scriptPath, objTree);
        } catch (const exception& e) {
            has_error = true;
            error_msg = "Error loading script: " + string(e.what());
        }
        return *this;
    }

    Handler& BuildCamera() {
        if (has_error) return *this;
        // 初始化图像缓冲区
        img = vector<MatrixXf>(3, MatrixXf(800, 800));
        // 设置摄像机
        camera.position = Vector3f(600.0f, 1100.0f, 600.0f);
        Vector3f lookAt = Vector3f(400.0f, -100.0f, -100.0f);
        camera.SetLookAt(lookAt);
        return *this;
    }

    Handler& Render() {
        if (has_error) return *this;
        try {
            auto start = chrono::high_resolution_clock::now();
            RayTracing::TraceRay(camera, objTree.Build(), img, 0, 100);
            auto stop = chrono::high_resolution_clock::now();
            auto duration = chrono::duration_cast<chrono::milliseconds>(stop - start);
            cout << "Time taken by traceRay: " << duration.count() << " milliseconds" << endl;
        } catch (const exception& e) {
            has_error = true;
            error_msg = "Rendering error: " + string(e.what());
        }
        return *this;
    }

    Handler& BuildResult() {
        if (has_error) return *this;
        imgout = ImageRGB(800, 800);
        for (int i = 0; i < 800; i++) {
            for (int j = 0; j < 800; j++) {
                imgout(i, j) = RGB(
                    min((int)(img[0](i, j) * 255), 255),
                    min((int)(img[1](i, j) * 255), 255),
                    min((int)(img[2](i, j) * 255), 255));
            }
        }
        return *this;
    }

    Handler& SaveResult() {
        if (has_error) return *this;
        string outputPath = "output.ppm";
        try {
            PPMWrite(outputPath, imgout);
            cout << "Rendering completed. Output saved to " << outputPath << endl;
        } catch (const exception& e) {
            has_error = true;
            error_msg = "Save error: " + string(e.what());
        }
        return *this;
    }

    // 设置脚本路径
    void SetScriptPath(const string& path) {
        scriptPath = path;
    }

    // 错误检查
    bool HasError() const { return has_error; }
    const string& GetError() const { return error_msg; }
};

int main(int argc, char* argv[]) {
    string default_file = "C:/Algo/Projects/Ray/src/ray_tracing/test.json";
    string scriptPath;

    // 参数处理
    if (argc < 2) {
        if (ifstream(default_file).good()) {
            scriptPath = default_file;
        } else {
            cerr << "Usage: " << argv[0] << " <file>" << endl;
            return 1;
        }
    } else {
        scriptPath = argv[1];
    }

    Handler h;
    h.SetScriptPath(scriptPath);
    h.PreCheck()
       .LoadScript()
       .BuildCamera()
       .Render()
       .BuildResult()
       .SaveResult();


    if (h.HasError()) {
        cerr << "Error: " << h.GetError() << endl;
        return 1;
    }

    return 0;
}