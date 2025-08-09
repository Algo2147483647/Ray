#ifndef IMAGE_IMAGE_H
#define IMAGE_IMAGE_H

#include <Eigen/Dense>
#include "ARGB.h"
#include "RGB.h"

typedef Eigen::Matrix<ARGB, Eigen::Dynamic, Eigen::Dynamic> ImageARGB;
typedef Eigen::Matrix<RGB, Eigen::Dynamic, Eigen::Dynamic> ImageRGB;

#endif // !IMAGE_H
