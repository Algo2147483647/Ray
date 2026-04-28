import numpy as np
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d import Axes3D
from PIL import Image
import os
import matplotlib
matplotlib.use('TkAgg')  # 使用TkAgg后端支持交互
def load_and_reshape_image(image_path, width, height, depth):
    """
    从图像文件加载数据并将其重塑为3D体积数据
    
    Args:
        image_path: 图像文件路径
        width: 原始数据的宽度 (x维度)
        height: 原始数据的高度 (y维度)
        depth: 原始数据的深度 (z维度)
    
    Returns:
        reshaped_data: 重塑后的3D数据 (width x height x depth x 3)
    """
    # 加载图像
    img = Image.open(image_path)
    img_array = np.array(img)
    
    # 确保图像是RGBA格式
    if len(img_array.shape) == 2:
        # 灰度图转RGBA
        img_array = np.stack([img_array, img_array, img_array, np.full_like(img_array, 255)], axis=-1)
    elif img_array.shape[2] == 3:
        # RGB转RGBA
        img_array = np.concatenate([img_array, np.full((img_array.shape[0], img_array.shape[1], 1), 255)], axis=-1)
    
    # 重塑数据为3D体积
    # 根据Go代码，数据被压平为(width, height*depth)的图像
    # 所以我们需要将其重塑为(width, height, depth)的3D体积
    reshaped_data = np.zeros((width, height, depth, 4), dtype=np.uint8)
    
    for z in range(depth):
        # 从压平的图像中提取每个z层
        layer = img_array[z*height:(z+1)*height, :width]
        reshaped_data[:, :, z, :] = layer
    
    return reshaped_data

def visualize_3d_volume(volume_data, threshold=50):
    """
    使用散点图在3D查看器中显示体积数据
    
    Args:
        volume_data: 4D数组 (width, height, depth, 4) - RGBA格式
        threshold: 透明度阈值，低于此值的像素将不显示
    """
    # 获取数据维度
    width, height, depth, _ = volume_data.shape
    
    # 创建坐标网格
    x, y, z = np.meshgrid(np.arange(width), np.arange(height), np.arange(depth), indexing='ij')
    
    # 将数据展平用于散点图
    x_flat = x.flatten()
    y_flat = y.flatten()
    z_flat = z.flatten()
    colors_flat = volume_data.reshape(-1, 4)
    
    # 计算透明度
    alphas = colors_flat[:, 3] / 255.0
    
    # 过滤掉透明度低于阈值的点
    mask = alphas >= (threshold / 255.0)
    x_filtered = x_flat[mask]
    y_filtered = y_flat[mask]
    z_filtered = z_flat[mask]
    colors_filtered = colors_flat[mask]
    alphas_filtered = alphas[mask]
    
    # 创建3D图形
    fig = plt.figure(figsize=(12, 9))
    ax = fig.add_subplot(111, projection='3d')
    
    # 归一化颜色值
    rgb_colors = colors_filtered[:, :3] / 255.0
    
    # 绘制散点图，每个点只有20%的透明度
    scatter = ax.scatter(x_filtered, y_filtered, z_filtered, 
                        c=rgb_colors, alpha=0.05, s=1)
    
    # 设置标签
    ax.set_xlabel('X Axis')
    ax.set_ylabel('Y Axis')
    ax.set_zlabel('Z Axis')
    ax.set_title('3D Volume Visualization')
    
    # 设置视角
    ax.view_init(elev=20, azim=45)
    
    # 根据项目规范设置坐标轴比例
    ax.set_box_aspect([1,1,1])
    
    plt.tight_layout()
    plt.show()

def main():
    # 图像路径
    image_path = r"/src-golang/output.png"
    
    # 检查图像文件是否存在
    if not os.path.exists(image_path):
        print(f"图像文件 {image_path} 不存在")
        return
    
    # 加载并重塑图像数据
    print("正在加载和重塑图像数据...")
    n = 200
    volume_data = load_and_reshape_image(image_path, n, n, n)
    
    # 显示3D体积
    print("正在显示3D体积数据...")
    visualize_3d_volume(volume_data)
    
    print("可视化完成")

if __name__ == "__main__":
    main()