import json
import numpy as np
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d import Axes3D
import matplotlib
from matplotlib.patches import Rectangle, Circle
import matplotlib.colors as mcolors
from matplotlib.collections import PatchCollection
import matplotlib.patches as mpatches
import matplotlib.lines as mlines

matplotlib.use('TkAgg')  # 使用TkAgg后端支持交互


# 从JSON文件加载数据
def load_scene_data(filename):
    with open(filename, 'r') as f:
        data = json.load(f)
    return data


# 绘制场景几何体
def draw_scene(ax, scene_data):
    print("正在绘制场景几何体...")

    # 为不同形状定义颜色
    shape_colors = {
        'cuboid': 'blue',
        'sphere': 'red'
    }

    # 创建图例元素
    legend_elements = [
        mpatches.Patch(color='blue', label='Cuboid'),
        mpatches.Patch(color='red', label='Sphere'),
        mlines.Line2D([], [], color='gray', linewidth=2, label='Ray Path'),
        mlines.Line2D([], [], marker='o', color='w', markerfacecolor='darkred', markersize=8, label='Ray Origin')
    ]

    # 绘制所有物体
    for obj in scene_data['objects']:
        shape = obj['shape']
        obj_id = obj['id']
        color = shape_colors.get(shape, 'green')

        # 绘制长方体
        if shape == 'cuboid':
            size = obj['size']
            position = np.array(obj['position'])
            half_size = np.array(size) / 2
            x = position[0]
            y = position[1]
            z = position[2]
            sx = half_size[0]
            sy = half_size[1]
            sz = half_size[2]

            # 定义长方体的12条边
            edges = [
                [[x - sx, x + sx], [y - sy, y - sy], [z - sz, z - sz]],  # 底部前边
                [[x + sx, x + sx], [y - sy, y + sy], [z - sz, z - sz]],  # 底部右边
                [[x + sx, x - sx], [y + sy, y + sy], [z - sz, z - sz]],  # 底部后边
                [[x - sx, x - sx], [y + sy, y - sy], [z - sz, z - sz]],  # 底部左边

                [[x - sx, x + sx], [y - sy, y - sy], [z + sz, z + sz]],  # 顶部前边
                [[x + sx, x + sx], [y - sy, y + sy], [z + sz, z + sz]],  # 顶部右边
                [[x + sx, x - sx], [y + sy, y + sy], [z + sz, z + sz]],  # 顶部后边
                [[x - sx, x - sx], [y + sy, y - sy], [z + sz, z + sz]],  # 顶部左边

                [[x - sx, x - sx], [y - sy, y - sy], [z - sz, z + sz]],  # 左前竖边
                [[x + sx, x + sx], [y - sy, y - sy], [z - sz, z + sz]],  # 右前竖边
                [[x + sx, x + sx], [y + sy, y + sy], [z - sz, z + sz]],  # 右后竖边
                [[x - sx, x - sx], [y + sy, y + sy], [z - sz, z + sz]]  # 左后竖边
            ]

            # 绘制所有边
            for edge in edges:
                ax.plot(edge[0], edge[1], edge[2], color=color, linewidth=1.5, alpha=0.7)

        # 绘制球体
        elif shape == 'sphere':
            radius = obj['r']
            position = np.array(obj['position'])
            # 在三个平面上绘制圆形来表示球体
            u = np.linspace(0, 2 * np.pi, 30)
            v = np.linspace(0, np.pi, 15)

            # XY平面上的圆 (Z=position[2])
            x_xy = position[0] + radius * np.cos(u)
            y_xy = position[1] + radius * np.sin(u)
            z_xy = np.full_like(x_xy, position[2])
            ax.plot(x_xy, y_xy, z_xy, color=color, linewidth=1.5, alpha=0.7)

            # XZ平面上的圆 (Y=position[1])
            x_xz = position[0] + radius * np.cos(u)
            z_xz = position[2] + radius * np.sin(u)
            y_xz = np.full_like(x_xz, position[1])
            ax.plot(x_xz, y_xz, z_xz, color=color, linewidth=1.5, alpha=0.7)

            # YZ平面上的圆 (X=position[0])
            y_yz = position[1] + radius * np.cos(u)
            z_yz = position[2] + radius * np.sin(u)
            x_yz = np.full_like(y_yz, position[0])
            ax.plot(x_yz, y_yz, z_yz, color=color, linewidth=1.5, alpha=0.7)

        elif shape == 'triangle':
            p1 = np.array(obj['p1'])
            p2 = np.array(obj['p2'])
            p3 = np.array(obj['p3'])

            edges = [
                [p1, p2],
                [p2, p3],
                [p3, p1]
            ]
            for edge in edges:
                points = np.array(edge)
                ax.plot(points[:, 0], points[:, 1], points[:, 2], color=color, linewidth=1.5, alpha=0.7)
    # 添加图例
    ax.legend(handles=legend_elements, loc='upper right')
    print("场景几何体绘制完成！")


# 绘制3D光线路径
def plot_ray_paths(ray_data, scene_data=None):
    # 创建3D图形
    fig = plt.figure(figsize=(14, 10))
    ax = fig.add_subplot(111, projection='3d')

    # 存储所有坐标点用于设置坐标轴范围
    all_points = []

    # 如果有场景数据，先绘制场景
    if scene_data:
        draw_scene(ax, scene_data)

    print(f"正在绘制 {len(ray_data)} 条光线路径...")

    # 处理每条光线
    for i, ray in enumerate(ray_data):
        # 解析数据
        origin = np.array(ray['start'])
        endpoint = np.array(ray['end'])
        color = ray['color']  # 使用原始颜色
        color = [min(c, 1.0) for c in color]

        # 收集坐标点
        all_points.append(origin)
        all_points.append(endpoint)

        # 绘制光线路径
        ax.plot(
            [origin[0], endpoint[0]],
            [origin[1], endpoint[1]],
            [origin[2], endpoint[2]],
            color=[max(c - 0.2, 0) for c in color],
            linewidth=0.5, alpha=0.7
        )

        # 标记起点（使用更深的颜色）
        start_color = [max(c - 0.2, 0) for c in color]  # 加深颜色
        ax.scatter(
            origin[0], origin[1], origin[2],
            color=start_color, s=10, depthshade=False, alpha=0.7
        )

    # 设置坐标轴范围
    if all_points:
        all_points = np.array(all_points)
        min_val = np.min(all_points, axis=0)
        max_val = np.max(all_points, axis=0)
        padding = np.max(max_val - min_val) * 0.15

        ax.set_xlim([min_val[0] - padding, max_val[0] + padding])
        ax.set_ylim([min_val[1] - padding, max_val[1] + padding])
        ax.set_zlim([min_val[2] - padding, max_val[2] + padding])

    # 添加标签
    ax.set_xlabel('X Axis', fontsize=12)
    ax.set_ylabel('Y Axis', fontsize=12)
    ax.set_zlabel('Z Axis', fontsize=12)
    ax.set_title('3D Scene with Ray Paths', fontsize=16)

    # 设置坐标轴比例为1:1:1
    ax.set_box_aspect([1,1,1])

    ax.view_init(elev=30, azim=-45)
    ax.grid(True, alpha=0.2)
    plt.tight_layout()
    plt.show()



# 主程序
if __name__ == "__main__":
    # 指定JSON文件路径
    scene_file = "../test.json"  # 场景描述文件
    ray_file = "../debug_traces.json"  # 光线路径文件

    # 加载场景数据
    scene_data = None
    try:
        scene_data = load_scene_data(scene_file)
        print(f"成功加载场景文件: {scene_file}")
    except FileNotFoundError:
        print(f"警告：场景文件 '{scene_file}' 未找到，将只绘制光线路径")
    except json.JSONDecodeError:
        print(f"警告：场景文件 '{scene_file}' 不是有效的JSON格式，将只绘制光线路径")
    except Exception as e:
        print(f"加载场景文件时发生错误: {str(e)}，将只绘制光线路径")

    # 加载光线数据并绘图
    try:
        ray_data = load_scene_data(ray_file)  # 使用相同的加载函数
        plot_ray_paths(ray_data, scene_data)
    except FileNotFoundError:
        print(f"错误：光线文件 '{ray_file}' 未找到")
    except json.JSONDecodeError:
        print(f"错误：光线文件 '{ray_file}' 不是有效的JSON格式")
    except KeyError as e:
        print(f"错误：JSON文件中缺少必要的键 - {str(e)}")
    except Exception as e:
        print(f"发生意外错误: {str(e)}")