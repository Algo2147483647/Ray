package camera

import (
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"image/color"
	"math"
	"src-golang/model/optics"
	"testing"
)

// TestCameraNDim_GenerateRay 测试CameraNDim的GenerateRay方法
func TestCameraNDim_GenerateRay(t *testing.T) {
	// 创建一个3D相机进行测试
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(3, []float64{1, 0, 0}), // X轴
		mat.NewVecDense(3, []float64{0, 1, 0}), // Y轴
		mat.NewVecDense(3, []float64{0, 0, 1}), // Z轴
	}
	camera.Width = []int{100, 100, 100}
	camera.FieldOfView = []float64{90, 90, 90}

	// 测试生成光线
	ray := camera.GenerateRay(nil, 50, 50, 50)

	// 检查光线是否正确初始化
	if ray == nil {
		t.Fatal("Expected ray to be generated, got nil")
	}

	// 检查光线原点是否正确
	if !mat.Equal(ray.Origin, camera.Position) {
		t.Errorf("Expected ray origin to match camera position")
	}

	// 检查光线方向是否已归一化
	directionNorm := mat.Norm(ray.Direction, 2)
	if math.Abs(directionNorm-1.0) > 1e-10 {
		t.Errorf("Expected direction to be normalized, got norm: %f", directionNorm)
	}

	// 检查颜色是否正确初始化
	expectedColor := mat.NewVecDense(3, []float64{1, 1, 1})
	if !mat.Equal(ray.Color, expectedColor) {
		t.Errorf("Expected color to be [1,1,1]")
	}

	t.Logf("Generated ray: Origin=%v, Direction=%v", ray.Origin.RawVector().Data, ray.Direction.RawVector().Data)
}

// TestCameraNDim_VisualizeRays 可视化生成的光线
func TestCameraNDim_VisualizeRays(t *testing.T) {
	// 创建一个3D相机进行测试
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 5})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(3, []float64{1, 0, 0}),  // X轴
		mat.NewVecDense(3, []float64{0, 1, 0}),  // Y轴
		mat.NewVecDense(3, []float64{0, 0, -1}), // Z轴（朝向原点）
	}
	camera.Width = []int{10, 10, 10}
	camera.FieldOfView = []float64{45, 45, 45}

	// 创建3D图
	p := plot.New()
	p.Title.Text = "CameraNDim Generated Rays"
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"

	// 创建点集用于显示相机位置
	cameraPoints := make(plotter.XYs, 1)
	cameraPoints[0].X = camera.Position.AtVec(0)
	cameraPoints[0].Y = camera.Position.AtVec(1)

	// 创建点集用于显示光线终点
	var rayPoints plotter.XYs

	// 创建线集用于显示光线
	var lines []plotter.XYs

	// 生成多个光线进行可视化
	for x := 0; x < 10; x += 3 {
		for y := 0; y < 10; y += 3 {
			for z := 0; z < 10; z += 3 {
				ray := camera.GenerateRay(&optics.Ray{}, x, y, z)

				// 计算光线终点（原点+方向*长度）
				rayEnd := mat.NewVecDense(ray.Direction.Len(), nil)
				rayEnd.AddScaledVec(ray.Origin, 3.0, ray.Direction)

				// 添加光线起点和终点到线集中
				line := make(plotter.XYs, 2)
				line[0].X = ray.Origin.AtVec(0)
				line[0].Y = ray.Origin.AtVec(1)
				line[1].X = rayEnd.AtVec(0)
				line[1].Y = rayEnd.AtVec(1)
				lines = append(lines, line)

				// 添加光线终点到点集中
				rayPoints = append(rayPoints, plotter.XY{X: rayEnd.AtVec(0), Y: rayEnd.AtVec(1)})
			}
		}
	}

	// 绘制相机位置
	cameraScatter, err := plotter.NewScatter(cameraPoints)
	if err != nil {
		t.Fatal(err)
	}
	cameraScatter.Color = color.RGBA{R: 255, A: 255} // 红色
	cameraScatter.Radius = vg.Points(5)

	// 绘制光线终点
	rayScatter, err := plotter.NewScatter(rayPoints)
	if err != nil {
		t.Fatal(err)
	}
	rayScatter.Color = color.RGBA{B: 255, A: 255} // 蓝色

	// 绘制光线
	for i, line := range lines {
		linePlot, err := plotter.NewLine(line)
		if err != nil {
			t.Fatal(err)
		}
		// 使用不同颜色显示光线
		linePlot.Color = plotutil.Color(i)
		p.Add(linePlot)
	}

	p.Add(cameraScatter, rayScatter)
	p.Legend.Add("Camera", cameraScatter)
	p.Legend.Add("Ray Endpoints", rayScatter)

	// 保存图像
	if err := p.Save(10*vg.Inch, 10*vg.Inch, "camera_ndim_rays.png"); err != nil {
		t.Fatal(err)
	}

	t.Log("Generated visualization: camera_ndim_rays.png")
}

// TestCameraNDim_4D 测试4维相机
func TestCameraNDim_4D(t *testing.T) {
	// 创建一个4D相机进行测试
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(4, []float64{0, 0, 0, 0})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		mat.NewVecDense(4, []float64{0, 0, 0, 1}),
	}
	camera.Width = []int{10, 10, 10, 10}
	camera.FieldOfView = []float64{90, 90, 90, 90}

	// 测试生成4D光线
	ray := camera.GenerateRay(nil, 5, 5, 5, 5)

	// 检查光线是否正确初始化
	if ray == nil {
		t.Fatal("Expected ray to be generated, got nil")
	}

	// 检查光线维度是否正确
	if ray.Origin.Len() != 4 {
		t.Errorf("Expected ray origin to be 4D, got %dD", ray.Origin.Len())
	}

	if ray.Direction.Len() != 4 {
		t.Errorf("Expected ray direction to be 4D, got %dD", ray.Direction.Len())
	}

	// 检查光线方向是否已归一化
	directionNorm := mat.Norm(ray.Direction, 2)
	if math.Abs(directionNorm-1.0) > 1e-10 {
		t.Errorf("Expected direction to be normalized, got norm: %f", directionNorm)
	}

	t.Logf("Generated 4D ray: Origin=%v, Direction=%v",
		ray.Origin.RawVector().Data,
		ray.Direction.RawVector().Data)
}

// TestCameraNDim_VisualizeRays4D 可视化4D相机生成的光线
// 由于4D空间无法直接可视化，我们投影到3D空间进行可视化
func TestCameraNDim_VisualizeRays4D(t *testing.T) {
	// 创建一个4D相机进行测试
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(4, []float64{0, 0, 0, 5}) // 在第4维上偏移
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),  // X轴
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),  // Y轴
		mat.NewVecDense(4, []float64{0, 0, 1, 0}),  // Z轴
		mat.NewVecDense(4, []float64{0, 0, 0, -1}), // W轴（朝向原点）
	}
	camera.Width = []int{10, 10, 10, 10}
	camera.FieldOfView = []float64{45, 45, 45, 45}

	// 创建3D图（将4D投影到3D进行可视化）
	p := plot.New()
	p.Title.Text = "4D CameraNDim Projected Rays"
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"

	// 创建点集用于显示相机位置（投影到3D）
	cameraPoints := make(plotter.XYs, 1)
	cameraPoints[0].X = camera.Position.AtVec(0)
	cameraPoints[0].Y = camera.Position.AtVec(1)

	// 创建点集用于显示光线终点
	var rayPoints plotter.XYs

	// 创建线集用于显示光线
	var lines []plotter.XYs

	// 生成多个光线进行可视化
	for x := 0; x < 10; x += 3 {
		for y := 0; y < 10; y += 3 {
			for z := 0; z < 10; z += 3 {
				for w := 0; w < 10; w += 3 {
					ray := camera.GenerateRay(&optics.Ray{}, x, y, z, w)

					// 计算光线终点（原点+方向*长度）
					rayEnd := mat.NewVecDense(ray.Direction.Len(), nil)
					rayEnd.AddScaledVec(ray.Origin, 3.0, ray.Direction)

					// 投影到前3个维度，再投影到XY平面用于可视化
					line := make(plotter.XYs, 2)
					line[0].X = ray.Origin.AtVec(0)
					line[0].Y = ray.Origin.AtVec(1)
					line[1].X = rayEnd.AtVec(0)
					line[1].Y = rayEnd.AtVec(1)
					lines = append(lines, line)

					// 添加光线终点到点集中
					rayPoints = append(rayPoints, plotter.XY{X: rayEnd.AtVec(0), Y: rayEnd.AtVec(1)})
				}
			}
		}
	}

	// 绘制相机位置
	cameraScatter, err := plotter.NewScatter(cameraPoints)
	if err != nil {
		t.Fatal(err)
	}
	cameraScatter.Color = color.RGBA{R: 255, A: 255} // 红色
	cameraScatter.Radius = vg.Points(5)

	// 绘制光线终点
	rayScatter, err := plotter.NewScatter(rayPoints)
	if err != nil {
		t.Fatal(err)
	}
	rayScatter.Color = color.RGBA{G: 255, A: 255} // 绿色

	// 绘制光线
	for i, line := range lines {
		linePlot, err := plotter.NewLine(line)
		if err != nil {
			t.Fatal(err)
		}
		// 使用不同颜色显示光线
		linePlot.Color = plotutil.Color(i)
		p.Add(linePlot)
	}

	p.Add(cameraScatter, rayScatter)
	p.Legend.Add("4D Camera", cameraScatter)
	p.Legend.Add("Ray Endpoints", rayScatter)

	// 保存图像
	if err := p.Save(10*vg.Inch, 10*vg.Inch, "camera_ndim_rays_4d.png"); err != nil {
		t.Fatal(err)
	}

	t.Log("Generated 4D visualization: camera_ndim_rays_4d.png")
}
