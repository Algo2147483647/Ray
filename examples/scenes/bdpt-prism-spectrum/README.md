# 牛顿式三棱柱投影实验

这个场景把经典牛顿棱镜实验布置成一间低调的历史光学暗房：白光经过狭缝和三棱柱后发生波长相关折射，最终投影到白色漫反射屏幕上。宽画幅主相机同时拍到投影屏、黄铜棱镜架、光阑、木质光学台和狭缝灯箱，光谱是叙事焦点，但不再悬浮在没有空间信息的纯黑背景中。

主视图包含一盏 2350 K 的镜外暖色摄影灯和一颗很小的 2200 K 悬挂实用灯，用来表现木材、黄铜、深蓝墙面及装置轮廓。艺术灯只存在于 `beauty.json`；诊断命令不加载该文件，因此统计测试天然保持为严格的单光源实验。

## 光路与布局

```text
6500 K 发光狭缝 → 准直光阑 → Cauchy 色散三棱柱 → 白色漫反射墙 → 相机
```

- 发光狭缝位于 `x=1.85`，提供连续可见光谱。
- 黑色光阑位于 `x≈0.2`，开口约为 `y=[0.145,0.22]`。
- 棱柱横截面顶点为 `(-0.5,-0.55)`、`(-0.5,0.55)`、`(-0.15,0)`。
- 棱柱使用理想 `specular_dielectric`，折射率为
  `n(λ)=1.35+0.025/λ²`，其中 `λ` 以微米计。这里使用高色散玻璃参数，确保墙面投影在有限分辨率下仍可辨认。
- 棱柱介质同时配置波长相关 `sigma_a`，在 400/500/600/700 nm 处分别为 `3.2/1.4/0.45/0.1`：蓝光吸收最强、红光最弱，用于验证 Light Tracing 在 delta 折射链内部正确累计 homogeneous Beer–Lambert 衰减。
- 投影屏位于 `x≈-4`，Lambert 反射率为 `0.92`。
- 相机位于装置斜前方；画面从左到右依次是屏幕光谱、带冷色边缘高光的透明棱镜、黄铜光阑和狭缝灯箱。
- 深蓝暗房墙、木地板、木质光学台、黄铜导轨和屏幕支架只负责空间叙事，均避开主光路。
- 棱镜的顶面和底面外侧使用极细的非发光轮廓杆，三条竖边使用 7 mm 半径的冷色轮廓杆。这是针对当前 `light_tracing(t=1)` 不直接投影 delta 表面的可读性辅助；验收相机只看屏幕，因此它们不进入统计画面。
- 场景没有环境光；未命中物体的相机射线仍为黑色，暗房由明确建模的暖色摄影灯照明。

中心光线的近似落点随波长单调变化：

```text
450 nm → y≈-0.122
550 nm → y≈-0.031
650 nm → y≈ 0.021
```

因此墙面上应出现一条沿 `z` 方向延伸、沿 `y` 方向从蓝紫到红色展开的竖直光谱带。

## 运行

场景像 `geometry-benchmark-matrix` 一样由多个 Studio JSON 合成：

- `scene.json`：共享光学装置、暗房和非发光舞台几何；
- `prism-control.json` / `prism-absorbing.json`：无吸收与带吸收的棱镜介质及封闭棱镜表面；
- `beauty.json`：艺术灯、主相机和正式输出；
- `diagnostic.json`：单光源诊断相机及默认测试规格；
- `verify-control.json` / `verify-absorbing.json`：两次诊断输出。

运行正式主视图：

```cmd
run.cmd
```

默认输出为 `320×160 @ 2048 spp`：

- `outputs/newton-prism-wall.png`
- `outputs/newton-prism-wall.bin`

快速构图预览：

```cmd
run.cmd --width 240 --height 120 --samples 128
```

`light_tracing` 同时追踪狭缝与艺术补光的光源子路径：前者穿过棱柱后在屏幕漫反射顶点执行 `t=1` 胶片投影，后者负责照亮舞台几何。即使低样本预览也应形成连续色带；提高到默认采样数主要用于平滑光谱和暗房照明噪声。

## 自动验收

```cmd
verify.cmd
```

`verify.cmd` 分别组合 control 与 absorbing JSON，并且两次都不加载 `beauty.json`。诊断相机保持 64° 垂直视场以覆盖屏幕高度，同时使用 48° 水平视场放大色带。命令完成两次 `light_tracing` 渲染后检查：

1. 蓝、绿、红波段都有有限正能量；
2. 三个波段在墙面上的横向质心严格单调；
3. 蓝红投影的总展开达到最低像素阈值；
4. control 与 absorbing 的三个波段质心保持在容差内；
5. 波段能量比严格满足 `blue < green < red`，且蓝光吸收强度足以排除“未应用 sigma_a”的假通过；

验收产物写入 `outputs/prism-spectrum-verify/`，其中包含 `control.*` 和 `absorbing.*` 两组结果，不会覆盖正式输出。

在默认 `120×80 @ 512 spp` 下，典型吸收/对照能量比约为 `blue=0.65`、`green=0.84`、`red=0.96`。这是 Monte Carlo 统计量，测试只要求稳定的波段顺序、足够强的蓝光衰减和合理的红光能量上界，不要求逐次运行完全一致。

## 积分器说明

这条路径属于典型焦散：

```text
light → delta transmission → delta transmission → diffuse wall → camera
```

普通 `bdpt` 目前仍会因为 `specular_dielectric` 和 `medium_boundary` 整场退回单向 path tracing。单向 PT 必须从墙面漫反射随机命中很窄的棱镜—光阑—狭缝方向域，因此即使 2048 spp 也主要表现为散粒噪声。

本场景改用独立的 `light_tracing` 积分器。它实现欧氏三维针孔相机的 `t=1` 策略：

- 按面积采样发光狭缝及余弦发射方向；
- 以光源传输模式穿过 Cauchy delta 折射链，并维护介质 IOR 栈；
- 在白墙等非 delta 顶点计算 BSDF；
- 使用针孔投影的面积到像素 Jacobian 将贡献 splat 到对应胶片像素。

该估计器对它所覆盖的光源子路径使用其真实采样 PDF 和投影 Jacobian，不使用 photon kernel，因此不会引入 photon mapping 的核半径偏差。它目前不是完整的全策略 BDPT：仅支持 Euclidean `Camera3D`，也还没有与 `t≥2`/纯相机策略进行全局 MIS；本场景的墙面焦散正好属于它高效覆盖的路径族。

### 积分器调度结构

运行时不在通用场景循环中用特殊 `if` 判断 Light Tracing。`IntegratorKind` 只负责配置名称，`NewSceneIntegrator` 在渲染开始前组合统一的 `SceneIntegrator`：

```text
SceneIntegrator
  ├─ RenderSession：校验、Film 初始化、累积与最终光谱转换
  └─ RenderDriver
       ├─ pixelDriver + pathTracingKernel / bdptKernel
       └─ splatDriver + lightTracingKernel
```

Path Tracing 和 BDPT 复用 `pixelDriver` 的 tile 调度，只提供各自的逐像素 kernel；Light Tracing kernel 只生成未归一化 `FilmSplat`，并发调度、总样本归一化、进度报告和线程安全写入都由通用 `splatDriver` 与 `FilmAccumulator` 负责。逐像素 driver 使用无锁独占写入，splat driver 使用按像素同步写入。

不支持投影的相机会返回明确错误，不会静默回退。旧配置字符串 `light_trace` 仅在解析边界作为兼容别名保留，运行时规范名称为 `light_tracing`。
