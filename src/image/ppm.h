#ifndef PPM_H
#define PPM_H

#include <iostream>
#include <fstream>
#include <vector>
#include <Eigen/Dense>
#include "../image.h"

void PPMRead(const std::string& fileName, ImageRGB& image) {
    std::ifstream file(fileName, std::ios::binary);
    if (!file.is_open()) {
        throw std::runtime_error("Cannot open file: " + fileName);
    }

    std::string header;
    int rows, cols, maxVal;
    file >> header >> cols >> rows >> maxVal;
    file.ignore(); // Ignore the newline character after the header

    if (header != "P6" || maxVal != 255) {
        throw std::runtime_error("Invalid PPM file format: " + fileName);
    }

    image.resize(rows, cols);
    file.read(reinterpret_cast<char*>(image.data()), image.size() * 3);

    if (!file) {
        throw std::runtime_error("Error reading PPM file: " + fileName);
    }

    file.close();
}

void PPMWrite(const std::string& fileName, const ImageRGB& image) {
    std::ofstream file(fileName, std::ios::binary);
    if (!file.is_open()) {
        throw std::runtime_error("Cannot open file: " + fileName);
    }

    file << "P6\n" << image.cols() << " " << image.rows() << "\n255\n";
    file.write(reinterpret_cast<const char*>(image.data()), image.size() * 3);

    if (!file) {
        throw std::runtime_error("Error writing PPM file: " + fileName);
    }

    file.close();
}

void PPMWrite(const std::string& fileName, const ImageARGB& image) {
    ImageRGB tmp(image.rows(), image.cols());

    for (int i = 0; i < image.rows(); ++i) {
        for (int j = 0; j < image.cols(); ++j) {
            tmp(i, j).B = static_cast<unsigned char>(image(i, j));
            tmp(i, j).G = static_cast<unsigned char>(image(i, j) >> 8);
            tmp(i, j).R = static_cast<unsigned char>(image(i, j) >> 16);
        }
    }

    PPMWrite(fileName, tmp);
}

void PPMWrite(const std::string& fileName, const Eigen::Matrix<unsigned char, Eigen::Dynamic, Eigen::Dynamic>& image) {
    std::ofstream file(fileName, std::ios::binary);
    if (!file.is_open()) {
        throw std::runtime_error("Cannot open file: " + fileName);
    }

    file << "P5\n" << image.cols() << " " << image.rows() << "\n255\n";
    file.write(reinterpret_cast<const char*>(image.data()), image.size());

    if (!file) {
        throw std::runtime_error("Error writing PPM file: " + fileName);
    }

    file.close();
}

void PPMWrite(const std::string& fileName, const Eigen::Matrix<float, Eigen::Dynamic, Eigen::Dynamic>& image) {
    Eigen::Matrix<unsigned char, Eigen::Dynamic, Eigen::Dynamic> tmp(image.rows(), image.cols());

    for (int i = 0; i < image.size(); ++i) {
        tmp(i) = static_cast<unsigned char>(image(i) * 255.0f);
    }

    PPMWrite(fileName, tmp);
}

#endif // PPM_H
