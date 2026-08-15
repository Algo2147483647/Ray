# 双向路径追踪（BDPT）技术方案

## 1. 目标与边界

BDPT 同时从相机和发光面生成子路径，并连接两条子路径上的顶点。相比只从相机出发的路径追踪，它能主动找到“小光源 → 镜面链 → 漫反射面 → 相机”一类低概率路径，主要改善间接光、焦散和被遮挡区域的收敛。

当前代码提供一个连续表面路径的第二阶段实现：

- 旧 `path` 积分器保持默认和行为兼容。
- `bdpt` 可通过场景或命令行显式选择。
- 支持 Euclidean 3D、RGB、hero wavelength 和 sampled wavelength。
- 支持 sphere、triangle、circle、cuboid 及其普通 bounded wrapper 作为有限面积光源。
- 生成相机和光源子路径，连接所有深度允许的连续表面端点。
- 对同一连续完整路径枚举全部已启用的 `s>=1,t>=2` 策略，并用相邻策略 PDF 比值的 ratio walk 计算 power MIS。
- 场景能力和功率加权面积光分布在渲染开始时准备一次，不再逐样本扫描对象树。
- 相机与光源子路径都按 `russian_roulette_depth` 执行 RR，并在存活后补偿吞吐量。
- delta 表面和 homogeneous `sigma_a` 介质边界不再触发整场回退；`light → delta 链 → diffuse → camera` 由独立 `t=1` film splat 估计。
- 没有可采样面积光、使用非 Euclidean 几何或非互易 BSDF 时，自动退回原路径追踪，并输出一次明确诊断。

当前阶段有意不宣称覆盖以下能力：

- 曲率空间中的 BDPT。Klein / spherical 场景需要测地距离、Jacobian 和曲空间可见性，不能套用 Euclidean `1/r²`。
- 参与介质散射顶点。homogeneous `sigma_a` 已通过段透射率进入吞吐量和连接，`sigma_s`/phase-function 顶点仍未实现。
- 与连续策略共同 MIS 的通用 `t=1`、`s=0` 和 camera endpoint importance；当前 delta 焦散 `t=1` 是独立估计器。
- delta/折射事件的正反离散策略密度及其 eta 修正；因此含 delta 的 `t>=2` 连续连接会被跳过。
- 任意隐式/参数曲面的精确面积采样、环境光和点光源端点。
- 非互易 BSDF 的反向 PDF 和 shading-normal Jacobian 修正。

这些限制均是显式能力边界；真正不支持的场景会带原因诊断地回退 `path`，不会静默使用错误的几何项。

## 2. 数学模型

一条完整光路写作

```text
x₀(light) — x₁ — … — xₛ₋₁  ⟷  yₜ₋₁ — … — y₁ — y₀(camera)
```

`s` 是光源子路径顶点数，`t` 是相机子路径顶点数。连接两个表面顶点时，Euclidean 3D 几何项为

```text
G(x, y) = V(x, y) |nₓ·ωxy| |nᵧ·ωyx| / ||x-y||²
```

非光源端点连接的贡献为

```text
Cₛ,ₜ = βL · fL · G · fC · βC · wₛ,ₜ
```

连接到光源根顶点时，用发射函数替换 `fL`：

```text
C₁,ₜ = Le · βL,root · G · fC · βC · w₁,ₜ
```

所有方向 PDF 在落到下一个表面时转换为面积 PDF：

```text
pA(x→y) = pω(x→y) |nᵧ·ωyx| / ||x-y||²
```

这是 MIS 比值可比较的前提。不能直接混合立体角 PDF、面积 PDF 和离散 delta 概率。

## 3. 组件设计

### 3.1 面积光源采样协议

`shape.SurfaceSampler` 提供：

```go
SampleSurface(u maths.Sample2D) (SurfaceSample, bool)
SurfaceArea() float64
```

`SurfaceSample.PDFArea` 明确声明采样密度相对于面积测度。当前以 `Area × 估算发光功率` 为光源选择权重，再按各 shape 的面积 PDF 采样表面，因此联合密度为：

```text
p(light, x) = Weight(light)/Weight(total) × pA(x | light)
```

当前使用线性 CDF；当光源数较多时可进一步升级为保存同一选择 PDF 的 alias table。

### 3.2 路径顶点

每个 `bdptVertex` 保存：

- 世界空间位置、几何法线和 shading frame；
- 指向路径前驱的局部方向 `WoLocal`；
- 完整 `ShadingContext` 和命中的对象/材质；
- 到达该顶点前的光谱吞吐量 `Beta`；
- 前向面积 PDF `PDFFwdArea`；
- 从该顶点继续采样的 PDF 和 delta 标记；
- 是否为光源端点。

几何法线用于面积 Jacobian 和 `G`；shading frame 用于 BSDF `Eval/PDF`。两者不可混用。

### 3.3 子路径生成

相机子路径：

1. 相机生成当前像素的 primary ray。
2. 求最近交点并创建表面交互。
3. 在到达顶点前记录 `Beta` 和面积 PDF。
4. 使用 radiance transport mode 采样 BSDF。
5. 更新 `β *= f |cosθ| / pω`。
6. 到达配置深度后执行 Russian roulette；存活路径将吞吐量除以存活概率。

光源子路径：

1. 按 `面积 × 估算发光功率` 选择可采样发光对象。
2. 均匀采样表面点。
3. 保存联合面积 PDF：

   ```text
   pA(light, x) = p(light) pA(x | light)
   ```

4. 对双面发光选择一侧并做 cosine hemisphere sampling：

   ```text
   pω = 1/2 |cosθ| / π
   β = Le |cosθ| / (pA pω)
   ```

5. 后续连续表面使用 importance transport mode 采样。

### 3.4 连接和遮挡

连接只发生在具有有限 BSDF 值的端点。非互易 BSDF 仍会整场回退；delta 可在子路径中采样，但含 delta 的连续 `t>=2` MIS 连接会跳过，焦散改由独立 `t=1` splat 进入胶片。连接射线使用 `[EPS, distance-EPS]`，避免把两个端点自身误判为遮挡物；连接段同时评估当前 homogeneous medium 的透射率。

当前仍为每个 `(s,t)` 建立连接，但 MIS 使用相邻策略密度比的 ratio walk，不再为每个连接重新枚举并遍历全部策略。连接为 `O(depth²)`，每个连接的权重为 `O(depth)`；原先接近 `O(depth⁴)` 的热点降为 `O(depth³)`。后续可继续增加：

- 最大连接深度；
- 路径顶点预分配；
- visibility batch；
- 可选的连接候选裁剪。

### 3.5 MIS

对包含 `m` 个表面端点、按光源到相机排列的完整路径，当前启用：

```text
s ∈ [1,m-1], t >= 2
```

每个策略密度包含光源选择/面积密度、光源侧方向到面积 Jacobian，以及相机侧方向到面积 Jacobian。统一使用：

```text
w_s = p_s² / Σ_k p_k²
```

未实现、方向 PDF 为零或不满足能力边界的策略不进入分母。实现使用 log-sum-exp 形式计算，自动测试对同一路径断言 `Σw=1`。

此前的相邻策略局部归一化已经移除。对三个或更多策略分别与邻居归一化，不能保证同一完整路径的权重和为 `1`。

当前已使用不组装完整路径的 ratio walk。要达到完整 Veach BDPT，下一步仍需：

1. 顶点同时保存 `pdfFwd`、`pdfRev` 和 endpoint 类型。
2. 连接前临时计算连接边两侧 reverse PDF。
3. 沿相机和光源子路径分别累乘 `pdfRev/pdfFwd`。
4. 跳过连接端为 delta 或方向不满足离散事件的策略。
5. 将所有有效策略的平方比值加入分母。

## 4. 光谱、介质和几何

RGB 模式下 `Beta` 是 RGB spectrum；hero/sampled 模式下它是单通道 sampled spectrum。波长由积分器在相机样本层选择，同一个 BDPT 样本的相机/光源子路径共享波长，避免把不同波长的两条路径错误连接。

参与介质散射支持仍必须增加 `MediumVertex`：

- 采样自由飞行距离；
- 保存 phase function、正反 PDF 和 transmittance；
- 连接边评估 `Tr(x,y)`；
- 处理相机、光源两侧的 medium stack；
- 对折射使用 radiance/importance 的 eta 修正。

曲空间支持必须由 `geometry.Geometry` 提供：

- 两点间可连接的测地线及多解策略；
- 测地距离；
- 端点切向量；
- exponential-map 面积 Jacobian；
- 沿测地线的遮挡查询。

在这些接口落地前，BDPT 对非 Euclidean 场景返回零并由上层保留显式能力约束；不应使用 Euclidean 距离项近似。

## 5. 配置与使用

场景：

```json
{
  "render": {
    "integrator": "bdpt",
    "samples": 256,
    "spectrum_mode": "hero_wavelength"
  }
}
```

命令行：

```bash
go -C engine run . --script ../examples/scenes/scene.json --integrator bdpt
```

可选值：

- `path`：原相机路径追踪，默认值；
- `bdpt`：双向路径追踪。

如果场景使用 participating media、非 Euclidean geometry 或发光隐式曲面，现阶段应选择 `path`。

## 6. 验证方案

现有自动测试覆盖：

- sphere / triangle / cuboid 面积公式；
- 表面样本落在几何体上且 `pA=1/Area`；
- 非 3D shape 不被错误声明为 Euclidean 面积光；
- 可见面积光与漫反射相机顶点连接后产生有限、非负、非零贡献；
- 没有可采样面积光时回退 path tracer；
- CLI 接受 `bdpt` 并拒绝未知积分器；
- 全仓 Go 回归测试。

建议增加的离线统计测试：

1. Cornell box：BDPT 与高 spp path tracing 的均值差异在置信区间内。
2. 小面积顶灯：固定时间比较方差。
3. 玻璃球焦散：检查 light-delta-diffuse-camera 路径。
4. 封闭遮挡：连接贡献严格为零。
5. 多灯面积比例：各灯被选择次数符合联合 PDF。
6. RGB / hero / sampled 三模式能量一致性。

## 7. 生产化路线

### M1：当前可运行基线

- 有限面积光采样；
- 相机子路径与 `s=1` 光源端点连接；
- 单策略权重 `1`，不使用不完整 MIS；
- RGB/光谱接入；
- 非 Euclidean/参与介质回退和精确贡献测试。

### M2：当前连续表面多策略 BDPT

- 相机/光源双子路径；
- 全部 `s>=1,t>=2` 连续表面连接；
- ratio-walk power MIS；
- 同路径策略权重和测试；
- 每次渲染缓存能力分析和功率加权光源分布；
- 相机/光源子路径 Russian roulette；
- homogeneous `sigma_a` 段透射率；
- delta 链之后的独立 `t=1` caustic splat；
- 非互易和非 Euclidean 场景带原因诊断地安全回退。

### M2.1：Veach 完整 BDPT

- Camera endpoint importance 与纳入统一 MIS 的通用 `t=1` film splat；
- 通用 `s=0` 与正反离散策略；
- 显式 `pdfFwd/pdfRev`、RR 和离散测度；
- delta/折射 eta/非互易校验；
- 多光源 alias table。

### M3：介质与曲空间

- medium vertex、phase function、segment transmittance；
- 曲空间测地连接和 Jacobian；
- 对不支持的 shape/geometry 在配置阶段给出错误，而非运行期降级。

### M4：性能

- arena/pool 化 path vertex；
- 批量 shadow rays；
- 连接深度与策略开关；
- 多光源 alias table；
- 扩展 benchmark：samples/s、有效连接率、shadow-ray 命中率、每策略方差。

## 8. 完成标准

“完整 BDPT”应同时满足：

- 所有启用策略都具有一致的路径测度；
- `pdfFwd/pdfRev` 包含光源选择、位置、方向、RR 和离散事件概率；
- delta 策略不会被错误连接；
- radiance / importance 传输和 eta 缩放成对正确；
- 所有策略 MIS 权重对同一路径求和为 1；
- RGB 与光谱模式无系统性能量偏差；
- 与参考积分器进行统计而非逐像素一致性验证；
- 不支持的几何、介质、光源类型在启动时可诊断。

## 9. 当前实现的数学与物理审查

### 9.1 结论

当前实现应定义为“欧氏 3D 连续表面 BDPT + 独立 delta-caustic `t=1` 估计器”，而不是完整的 Veach BDPT。

在以下限制同时成立时，它对最大深度以内的路径积分基本无偏：

- 场景是三维 Euclidean 空间；
- 光源是有限、可按面积采样的双面面积光；
- 连续 MIS 路径只包含 Lambert、rough conductor 等连续、互易的反射 BSDF；
- delta 链仅由独立 `t=1` 路径族覆盖，不参与连续多策略 MIS；
- homogeneous 介质只包含可由段透射率表达的吸收，不包含参与散射顶点；
- 不包含非互易散射或曲空间连接；
- 将 `MaxRayLevel` 视为积分目标的一部分。

Russian roulette 消除了仅因 RR 提前终止造成的偏差，但 `MaxRayLevel` 仍定义了有限的最大路径长度。非互易或非 Euclidean 场景会带原因诊断地整体退回单向 path tracing；普通 delta 表面和 homogeneous 吸收边界不会再触发回退。

### 9.2 成立的公式

连续连接边使用

```text
G(x,y) = V(x,y) |n_x·ω_xy| |n_y·ω_yx| / ||x-y||²
```

非光源端点连接贡献为

```text
C_s,t = βL fL G fC βC w_s,t
```

光源根端点连接使用

```text
C_1,t = Le βL,root G fC βC w_1,t
```

方向密度到面积密度的转换为

```text
pA(y|x) = pω(ω_xy|x) |n_y·ω_yx| / ||x-y||²
```

实现中的吞吐量、连接几何项与 MIS 密度使用了同一面积测度。对同一条完整路径，所有启用策略使用全局 power heuristic：

```text
w_s = p_s² / Σ_k p_k²
```

因此在受支持策略族中 `Σ_s w_s = 1`。每个策略的估计量为 `F/p_s`，所有策略期望之和为该有限路径域上的路径积分。

光源按面积与估算发光功率的乘积选择，再使用对象返回的面积 PDF。联合密度为

```text
p(light,x) = Weight(light)/Weight(total) × pA(x|light)
```

该联合密度同时进入根顶点吞吐量，因此仍保持无偏；功率加权可降低多灯场景的方差。

### 9.3 不能扩展宣称无偏的部分

1. **仍有最大深度截断。** 相机和光源子路径已执行 Russian roulette，但仍受 `MaxRayLevel` 硬上限约束；因此目标仍是有限最大长度的路径空间。
2. **策略族不完整。** 当前没有通用 `s=0`、与其他策略统一 MIS 的 `t=1`、camera endpoint importance 和 delta 离散策略。现有 `t=1` 只负责光源侧已采样 delta 链后的 caustic splat。
3. **参与介质散射未进入 BDPT。** homogeneous `sigma_a` 已通过 segment transmittance 支持，但仍缺少 medium vertex、phase function；折射链还缺少用于完整双向 MIS 的 eta 与正反离散 PDF。
4. **三维假设未被类型契约完全表达。** 光源方向采样使用 3D cosine hemisphere，端点 PDF 使用 `|cos|/(2π)`，连接使用 `1/r²`。能力门禁目前只检查 Euclidean geometry；若将来 N 维 Euclidean shape 实现 `SurfaceSampler`，这些公式不能直接复用。
5. **发光模型隐含双面 Lambertian 方向分布。** 当前 emitter 的 `Emit` 都不依赖方向，因此与实现一致；若增加单面或方向性 emitter，必须同时增加方向分布、PDF 和能力声明。
6. **delta 覆盖仍是专用路径族。** delta 不再导致全场回退，但当前只保证光源侧 delta 链到漫反射顶点的 `t=1` 焦散估计。其他 delta 连接拓扑不能宣称由完整 BDPT MIS 覆盖。

### 9.4 当前验证的解释

现有自动测试验证了：

- 面积光能够连接到相机子路径；
- 基础连接贡献有限、非负且符合一个简单解析值；
- 全局 MIS 权重形成单位分割；
- delta 与 homogeneous 吸收不会触发能力门禁；
- delta 路径不进入连续 MIS，且能生成 `t=1` caustic splat；
- 多光源选择权重包含估算功率；
- RR 深度配置生效；
- 不支持的非 Euclidean 传输触发能力门禁。

这些测试能够发现公式和策略归一化的局部回归，但尚不能证明统计无偏。还需要：

1. 解析直接光场景的多随机种子均值与置信区间；
2. Cornell box 中 BDPT 与高样本参考积分器的线性能量比较；
3. 按 `(s,t)` 分离策略后验证各策略期望及合并期望；
4. 多光源面积/功率差异测试；
5. RGB、hero wavelength、sampled wavelength 的能量一致性测试；
6. 明确断言实际采用的是 BDPT 还是 path fallback。

棱镜光谱示例包含 `specular_dielectric`、homogeneous `sigma_a` 和漫反射光谱卡，现已成为 `light → 棱镜 delta 链 → 光谱卡 → camera` 的 BDPT `t=1` 回归场景。control/absorbing 两组正式测试会同时检查色散质心、跨度、参考一致性和分波段能量比；运行时无 path fallback。

### 9.5 工程质量判断

- 数学骨架：良好；
- 受限域内正确性：较好；
- 完整 BDPT 覆盖度：中低，delta 焦散已有专用 `t=1`，但尚无完整离散 MIS；
- 方差效率：较好，光源按面积与估算功率选择，子路径启用 RR；
- 性能：场景扫描移出样本循环；连接为 `O(depth²)`，每连接 ratio walk 为 `O(depth)`；
- 验证强度：已有单元、深度 benchmark 和棱镜 control/absorbing 统计能量回归。

下一阶段应优先补齐显式 `pdfFwd/pdfRev`、delta/eta 离散 MIS、参与介质顶点和连续表面参考能量统计，再做路径 arena、shadow-ray batch 与多光源 alias table。
