#ifndef IMAGE_RGB_H
#define IMAGE_RGB_H

#include "argb.h"

struct RGB {
	INT8U R, G, B;

    RGB() { ; }
	RGB(INT8U R, INT8U G, INT8U B) : R(R), G(G), B(B) {}

    void SetColor(INT8U newR, INT8U newG, INT8U newB) {
        R = newR;
        G = newG;
        B = newB;
    }
};

#endif