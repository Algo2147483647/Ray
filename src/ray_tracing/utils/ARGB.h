#ifndef RGB_H
#define RGB_H

#include <math.h>

#define INT8U unsigned char
#define ARGB  unsigned int


static ARGB GetARGB(INT8U R, INT8U G, INT8U B, INT8U A = 0xFF) {
	return ARGB(B) << 0 |
		ARGB(G) << 8 |
		ARGB(R) << 16 |
		ARGB(A) << 24;
}

static ARGB GetARGB(double R, double G, double B, double A = 1) {
	return
		(ARGB)(B * 0xFF) << 0 |
		(ARGB)(G * 0xFF) << 8 |
		(ARGB)(R * 0xFF) << 16 |
		(ARGB)(A * 0xFF) << 24;
}

static ARGB mul(ARGB a, double r) {
	return
		((ARGB)((a & 0x0000FF) * r) & 0x0000FF) |
		((ARGB)((a & 0x00FF00) * r) & 0x00FF00) |
		((ARGB)((a & 0xFF0000) * r) & 0xFF0000);
}

static ARGB Alpha(ARGB a, ARGB b, double r) {
	return
		((ARGB)((a & 0x0000FF) * r + (b & 0x0000FF) * (1 - r)) & 0x0000FF) |
		((ARGB)((a & 0x00FF00) * r + (b & 0x00FF00) * (1 - r)) & 0x00FF00) |
		((ARGB)((a & 0xFF0000) * r + (b & 0xFF0000) * (1 - r)) & 0xFF0000);
}

#endif