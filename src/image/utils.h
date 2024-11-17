#ifndef SPECTRUM_H
#define SPECTRUM_H

#include <string>
#include "Color.h"

namespace Graphics {
	inline ARGB spectrum(double index, const string model) {
		double A = 1, R = 1, G = 1, B = 1;
		double a = index, b = 1 - a;

		if(model == "rainbow")
			HSV2RGB(index * 360.0, 1.0, 1.0, R, G, B);

		return RGB::to_ARGB(R, G, B, A);
	}

    // Channel Blending Functions
    INT8U BlendNormal(INT8U A, INT8U B) {
        return A;
    }

    INT8U BlendLighten(INT8U A, INT8U B) {
        return (B > A) ? B : A;
    }

    INT8U BlendDarken(INT8U A, INT8U B) {
        return (B > A) ? A : B;
    }

    INT8U BlendMultiply(INT8U A, INT8U B) {
        return ((ARGB)A * B) / 0xFF;
    }

    INT8U BlendAverage(INT8U A, INT8U B) {
        return (A + B) / 2;
    }

    INT8U BlendAdd(INT8U A, INT8U B) {
        return min(0xFF, A + B);
    }

    INT8U BlendSubtract(INT8U A, INT8U B) {
        return (A + B < 0xFF) ? 0 : (A + B - 0xFF);
    }

    INT8U BlendDifference(INT8U A, INT8U B) {
        return abs(A - B);
    }

    INT8U BlendNegation(INT8U A, INT8U B) {
        return 0xFF - abs(0xFF - A - B);
    }

    INT8U BlendScreen(INT8U A, INT8U B) {
        return 0xFF - (((0xFF - A) * (0xFF - B)) >> 8);
    }

    INT8U BlendExclusion(INT8U A, INT8U B) {
        return A + B - 2 * A * B / 0xFF;
    }

    INT8U BlendOverlay(INT8U A, INT8U B) {
        return (B < 0x80) ? (2 * A * B / 0xFF) : (0xFF - 2 * (0xFF - A) * (0xFF - B) / 0xFF);
    }

    INT8U BlendSoftLight(INT8U A, INT8U B) {
        return (B < 0x80) ? (2 * ((A >> 1) + 64)) * ((float)B / 0xFF) : (0xFF - (2 * (0xFF - ((A >> 1) + 64)) * (float)(0xFF - B) / 0xFF));
    }

    INT8U BlendHardLight(INT8U A, INT8U B) {
        return BlendOverlay(B, A);
    }

    INT8U BlendColorDodge(INT8U A, INT8U B) {
        return (B == 0xFF) ? B : min(0xFF, ((A << 8) / (0xFF - B)));
    }

    INT8U BlendColorBurn(INT8U A, INT8U B) {
        return (B == 0) ? B : max(0, (0xFF - ((0xFF - A) << 8) / B));
    }

    INT8U BlendLinearDodge(INT8U A, INT8U B) {
        return BlendAdd(A, B);
    }

    INT8U BlendLinearBurn(INT8U A, INT8U B) {
        return BlendSubtract(A, B);
    }

    INT8U BlendLinearLight(INT8U A, INT8U B) {
        return (B < 0x80) ? BlendLinearBurn(A, (2 * B)) : BlendLinearDodge(A, (2 * (B - 0x80)));
    }

    INT8U BlendVividLight(INT8U A, INT8U B) {
        return (B < 0x80) ? BlendColorBurn(A, (2 * B)) : BlendColorDodge(A, (2 * (B - 0x80)));
    }

    INT8U BlendPinLight(INT8U A, INT8U B) {
        return (B < 0x80) ? BlendDarken(A, (2 * B)) : BlendLighten(A, (2 * (B - 0x80)));
    }

    INT8U BlendHardMix(INT8U A, INT8U B) {
        return (BlendVividLight(A, B) < 0x80) ? 0 : 0xFF;
    }

    INT8U BlendReflect(INT8U A, INT8U B) {
        return (B == 0xFF) ? B : min(0xFF, (A * A / (0xFF - B)));
    }

    INT8U BlendGlow(INT8U A, INT8U B) {
        return BlendReflect(B, A);
    }

    INT8U BlendPhoenix(INT8U A, INT8U B) {
        return min(A, B) - max(A, B) + 0xFF;
    }

    INT8U BlendAlpha(INT8U A, INT8U B, float O) {
        return O * A + (1 - O) * B;
    }

    template <typename Func>
    INT8U BlendAlphaF(INT8U A, INT8U B, Func F, float O) {
        return BlendAlpha(F(A, B), A, O);
    }

}
#endif