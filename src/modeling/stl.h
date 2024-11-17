#ifndef GRAPHICS_IO_H
#define GRAPHICS_IO_H

#include <vector>
#include "Image.h"

namespace Graphics {
/******************************************************************************
*					.STL 文件编码/解码
*	[格式]:
		[1] 头			(80B)
		[2] 三角面个数	(4B)
		[3] 对每个三角面(50B)(循环)
			[3.1] 法向量(3x4B)
			[3.2] 顶点1 (3x4B)
			[3.3] 顶点2 (3x4B)
			[3.4] 顶点3 (3x4B)
			[3.5] 属性	(2B)
******************************************************************************/

    inline void stlRead(const char* fileName, Mat<float>& faceVec, Mat<float>& p1, Mat<float>& p2, Mat<float>& p3, vector<short>& attribute) {
        FILE* fi = fopen(fileName, "rb");
        unsigned char head[80];
        unsigned int  N;
        float p[12];

        fread(head, 80, 1, fi);
        fread(&N,    4, 1, fi);

        faceVec.resize(3, N);
        p1.resize(3, N);
        p2.resize(3, N);
        p3.resize(3, N);
        attribute.resize(N);

        for (int i = 0; i < N; i++) {
            fread(p, 12 * 4, 1, fi);
            fread(attribute.data() + i, 2, 1, fi);
            faceVec	(0, i) = p[0];
            faceVec	(1, i) = p[1];
            faceVec	(2, i) = p[2];

            p1(0, i) = p[3];
            p1(1, i) = p[4];
            p1(2, i) = p[5];
            p2(0, i) = p[6];
            p2(1, i) = p[7];
            p2(2, i) = p[8];
            p3(0, i) = p[9];
            p3(1, i) = p[10];
            p3(2, i) = p[11];
        }
        fclose(fi);
    }

    inline void stlWrite(
        const char* fileName,
        const char* head,
        Mat<float>& faceVec,
        Mat<float>& p1,
        Mat<float>& p2,
        Mat<float>& p3,
        vector<short>& attribute
    ) {
        FILE* fo = fopen(fileName, "wb");
        unsigned int  N = p1.cols();
        fwrite(head, 80, 1, fo);
        fwrite(&N,    4, 1, fo);

        for (int i = 0; i < N; i++) {
            fwrite(&faceVec(0, i), 4, 1, fo);
            fwrite(&faceVec(1, i), 4, 1, fo);
            fwrite(&faceVec(2, i), 4, 1, fo);

            fwrite(&p1(0, i), 4, 1, fo);
            fwrite(&p1(1, i), 4, 1, fo);
            fwrite(&p1(2, i), 4, 1, fo);
            fwrite(&p2(0, i), 4, 1, fo);
            fwrite(&p2(1, i), 4, 1, fo);
            fwrite(&p2(2, i), 4, 1, fo);
            fwrite(&p3(0, i), 4, 1, fo);
            fwrite(&p3(1, i), 4, 1, fo);
            fwrite(&p3(2, i), 4, 1, fo);

            fwrite(attribute.data() + i, 2, 1, fo);
        }
        fclose(fo);
    }

    inline void stlWrite(const char* fileName, vector<vector<float>>& triangleSet) {
        int n = triangleSet.size();
        char head[80];
        Mat<float>
            faceVec(3, n),
            p1(3, n),
            p2(3, n),
            p3(3, n);
        vector<short> attribute(n);

        for (int i = 0; i < n; i++) {
            p1(0, i) = triangleSet[i][0];
            p1(1, i) = triangleSet[i][1];
            p1(2, i) = triangleSet[i][2];
            p2(0, i) = triangleSet[i][3];
            p2(1, i) = triangleSet[i][4];
            p2(2, i) = triangleSet[i][5];
            p3(0, i) = triangleSet[i][6];
            p3(1, i) = triangleSet[i][7];
            p3(2, i) = triangleSet[i][8];
        }

        stlWrite(fileName, head, faceVec, p1, p2, p3, attribute);
    }

}
#endif