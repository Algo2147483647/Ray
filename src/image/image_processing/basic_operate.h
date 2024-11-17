#ifndef IMAGE_PROCESSING_BASIC_OPERATE_H
#define IMAGE_PROCESSING_BASIC_OPERATE_H

#include "Image.h"

/************************************************
*
*				Basic Operate
*
************************************************/

namespace ImageProcessing {
	/*
	 * ��ֵ�� : ������ֵ����ͼ���Ϊ���ڰ�ͼ
	 */
	Mat<float>& Binarization(Mat<float>& in, Mat<float>& out, double threshold = 0.5) {
		return binarization(out, in, threshold);
	}

	/*
	 * ���� : ������ɫ�����䲹ɫ InvImage = 1 - Image
	 */
	Mat<float>& Inv(Mat<float>& in, Mat<float>& out) {
		out.zero(in.rows, in.cols);

		for (int i = 0; i < in.size(); i++)
			out(i) = 1 - in(i);

		return out;
	}

	void Inv(Mat<float>* in, Mat<float>* out, int N) {
		for (int i = 0; i < N; i++)
			Invert(in[i], out[i]);
	}

	/*
	 * ת�Ҷ�ͼ : ��ͨ��(RGB)��Ȩ�ϲ�Ϊ�Ҷ�һͨ�� Gray = 0.3 R + 0.59 G + 0.11 B
	 */
	Mat<float>& Gray(Mat<float>* in, Mat<float>& out, double Rk = 0.3, double Gk = 0.59, double Bk = 0.11) {
		out.zero(in[0].rows, in[0].cols);
		Mat<float> t;
		add(out, out, mul(t, Rk / (Rk + Gk + Bk), in[0]));
		add(out, out, mul(t, Gk / (Rk + Gk + Bk), in[1]));
		add(out, out, mul(t, Bk / (Rk + Gk + Bk), in[2]));
		return out;
	}

	Mat<float>& Gray(Mat<float>* in, Mat<float>& out, double* rate, int N) {
		out.zero(in[0].rows, in[0].cols);
		Mat<float> t;

		for (int i = 0; i < N; i++)
			add(out, out, mul(t, rate[i], in[i]);

		return out;
	}

}


#endif