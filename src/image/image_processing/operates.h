#ifndef IMAGE_PROCESSING_OTHER_OPERATE_H
#define IMAGE_PROCESSING_OTHER_OPERATE_H

#include "BasicMachineLearning.h"

namespace ImageProcessing {

	/*
	 *			��ɫ����
	 *	[Ŀ��]: �򻯾���ͼ���е�ɫ��.
	 *	[�㷨]: K-Mean��ֵ����
	 */
	void K_Mean(Mat<float>& x, int K, int TimesMax, Mat<float>& Center, Mat<int>& Cluster, Mat<int>& Cluster_Cur) {
		int Dimension = x.rows, N = x.cols;

		Center.zero(Dimension, K);
		Cluster.zero(K, N); 
		Cluster_Cur.zero(K, 1);

		//[1]
		for (int i = 0; i < K; i++) {
			int index = rand() % N;
			for (int dim = 0; dim < Dimension; dim++)
				Center(dim, i) = x(dim, index);
		}

		//[2]
		int Times = 0;
		
		while (true) {
			if (Times++ > TimesMax) return;
			
			//[3]
			Cluster.clean(); 
			Cluster_Cur.clean();

			//[4] 
			for (int i = 0; i < N; i++) {	
				Mat<float> d(1, K);

				for (int j = 0; j < K; j++)
					for (int dim = 0; dim < Dimension; dim++)
						d[j] += (x(dim, i) - Center(dim, j)) * (x(dim, i) - Center(dim, j));

				//[5]
				int index; 
				d.min(index);
				Cluster(index, Cluster_Cur[index]++) = i;
			}

			//[6] 
			Mat<float> CenterTemp(Dimension, K);
			for (int i = 0; i < K; i++) {
				for (int dim = 0; dim < Dimension; dim++) {

					for (int j = 0; j < Cluster_Cur[i]; j++) 
						CenterTemp(dim, i) += x(dim, Cluster(i, j));

					CenterTemp(dim, i) /= Cluster_Cur[i];
				}
			}

			//[7] 
			bool flag = 1;
			for (int i = 0; i < Dimension * K; i++) {
				if (CenterTemp[i] != Center[i]) { flag = 0; break; }
			}
			
			if (flag) return;								//[9]
			else {
				free(Center.data); 
				Center.data = CenterTemp.data; 
				CenterTemp.data = NULL;
			}
		}
	}

	Mat<float>* ColorCluster(Mat<float>* in, Mat<float>* out, int K = 3, int TimesMax = 0x7FFFFFFF) {
		// Process in & out
		Mat<float> data(3, in[0].size());
		for (int k = 0; k < 3; k++)
			for (int i = 0; i < in[0].rows; i++)
				for (int j = 0; j < in[0].cols; j++)
					data(k, i * in[0].cols + j) = (in[k])(i, j);

		for (int k = 0; k < 3; k++)
			out[k].zero(in[0].rows, in[0].cols);

		// Color Cluster
		Mat<float> Center;
		Mat<int> Cluster, Cluster_Cur;

		K_Mean(data, K, Center, Cluster, Cluster_Cur, TimesMax);
		for (int i = 0; i < K; i++)
			for (int j = 0; j < Cluster_Cur[i]; j++)
				for (int dim = 0; dim < 3; dim++)
					(out[dim])(Cluster(i, j) / out[0].cols, Cluster(i, j) % out[0].cols) = Center(dim, i);
		return out;
	}

	/*
	 *								��������
	 */
	double SobelKernelTmp[] = {
		-1,0,1,
		-2,0,2,
		-1,0,1
	};
	Mat<float> SobelKernel(3, 3, SobelKernelTmp);

	/*
	 *				��Ե���
	 *	[Ŀ��]: ��ʶ����ͼ�������ȱ仯���Եĵ�.
	 *	[��ʽ]: EdgeImage = Conv(Image , SobelKernel)
	 */
	Mat<float>& EdgeDetection(Mat<float>& in, Mat<float>& out) {
		Mat<float> out_x, out_y;
		conv(out_x, in, SobelKernel, 1);
		conv(out_y, in, SobelKernel.transpose(out_y), 1);

		out.zero(in);
		for (int i = 0; i < in.size(); i++)
			out[i] = sqrt(out_x[i] * out_x[i] + out_y[i] * out_y[i]);
		return out;
	}

	/*
	 *		����Ҷ�任
	 *	[Ŀ��]: תƵ��ͼ��.
	 */
	Mat<float>& FourierTransform(Mat<float>& in, Mat<float>& out) {
		return out;
	}

	Mat<float>& InvFourierTransform(Mat<float>& in, Mat<float>& out) {
		return out;
	}

	/*
	 *				Gauss �˲�
	 * [����]: in: ����ԭͼ out: ���ͼ��  size: �˵Ĵ�С  sigma: ��̬�ֲ���׼��
	 */
	Mat<float>& GaussFilter(Mat<float>& in, int size, float sigma, Mat<float>& out) {
		if (size <= 0 || sigma == 0)
			return out;

		Mat<float> GaussKernel(size, size);
		for (int y = 0; y < size; y++)
			for (int x = 0; x < size; x++)
				GaussKernel(x, y) = 1 / (2 * M_PI * sigma * sigma) * exp(-(pow(x - size / 2, 2) + pow(y - size / 2, 2)) / (2 * sigma * sigma));
		return out.conv(in, GaussKernel *= 1 / GaussKernel.sum(), 1);
	}

	/*
	 *								ֱ��ͼ
	 * [Ŀ��]: ͳ��[0,255]���ȵ����ظ����ֲ�.
	 */
	Mat<int>& Histograms(Mat<float>& in, Mat<float>& out) {
		out.zero(0xFF);
		for (int i = 0; i < in.size(); i++)
			out[(unsigned char)(in[i] * 0xFF)]++;
	}

}

#endif