# 双向路径追踪（BDPT）技术方案

## 1. 目标与边界

BDPT 同时从相机和发光面生成子路径，并连接两条子路径上的顶点。相比只从相机出发的路径追踪，它能主动找到“小光源 → 镜面链 → 漫反射面 → 相机”一类低概率路径，主要改善间接光、焦散和被遮挡区域的收敛。

当前代码提供一个连续表面路径的第二阶段实现：

- 旧 `path` 积分器保持默认和行为兼容。
- `bdpt` 可通过场景或命令行显式选择。
- 支持 Euclidean 3D、RGB、hero wavelength 和 sampled wavelength。
- 支持 sphere、triangle、circle、cuboid 及其普通 bounded wrapper 作为有限面积光源。
- 生成相机和光源子路径，连接所有深度允许的连续表面端点。
- 对同一完整路径枚举全部已启用的 `s>=1,t>=2` 策略，并以统一 power-heuristic 分母归一化。
- 路径密度和 MIS 在 log domain 计算，避免长路径概率乘积下溢。
- 没有可采样面积光、使用非 Euclidean 几何、介质边界、delta 或非互易 BSDF 时，自动退回原路径追踪。
- BDPT 子路径当前关闭 Russian roulette；完整反向 RR 密度加入前，仅由最大深度截断。

当前阶段有意不宣称覆盖以下能力：

- 曲率空间中的 BDPT。Klein / spherical 场景需要测地距离、Jacobian 和曲空间可见性，不能套用 Euclidean `1/r²`。
- 参与介质中的双向连接。当前 BDPT 顶点只表示表面，介质吸收/散射仍应使用 `path`。
- `t=1` 的 light tracing film splat 和参与同一 MIS 的通用 `s=0` 相机命中发光策略。
- delta/折射事件的离散策略密度及其 eta 修正。
- 任意隐式/参数曲面的精确面积采样、环境光和点光源端点。
- 非互易 BSDF 的反向 PDF 和 shading-normal Jacobian 修正。

这些限制均是显式能力边界；不满足条件时应保持 `path`，而不是静默使用错误的几何项。

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

`SurfaceSample.PDFArea` 明确声明采样密度相对于面积测度。光源先按面积选对象，再在对象表面均匀采样，因此联合密度简化为：

```text
p(light, x) = Area(light)/Area(total) × 1/Area(light)
            = 1/Area(total)
```

后续可将对象选择升级为“面积 × 估算功率”的 alias table。升级后必须同时保存选择 PDF，不能继续假设 `1/Area(total)`。

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
6. 当前按配置最大深度终止；暂不执行 Russian roulette。

光源子路径：

1. 按面积选择可采样发光对象。
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

连接只发生在具有有限 BSDF 值的端点。当前能力门禁会让包含 delta、介质边界或非互易 BSDF 的场景整场回退 `path`。连接射线使用 `[EPS, distance-EPS]`，避免把两个端点自身误判为遮挡物。

当前实现为每个 `(s,t)` 建立连接，并为该完整路径重新评估所有启用策略密度。路径深度较小时实现清晰可靠；优化前最坏复杂度高于标准 ratio walk。后续应增加：

- 最大连接深度；
- 路径顶点预分配；
- visibility batch；
- 光源分布缓存；
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

当前直接重算策略密度；生产级优化的下一步是改为 Veach ratio walk：

1. 顶点同时保存 `pdfFwd`、`pdfRev` 和 endpoint 类型。
2. 连接前临时计算连接边两侧 reverse PDF。
3. 沿相机和光源子路径分别累乘 `pdfRev/pdfFwd`。
4. 跳过连接端为 delta 或方向不满足离散事件的策略。
5. 将所有有效策略的平方比值加入分母。

## 4. 光谱、介质和几何

RGB 模式下 `Beta` 是 RGB spectrum；hero/sampled 模式下它是单通道 sampled spectrum。波长由积分器在相机样本层选择，同一个 BDPT 样本的相机/光源子路径共享波长，避免把不同波长的两条路径错误连接。

介质支持必须增加 `MediumVertex`：

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
- log-domain 全策略 power MIS；
- 同路径策略权重和测试；
- delta、介质边界和非互易场景安全回退。

### M2.1：Veach 完整 BDPT

- Camera endpoint importance 与 `t=1` film splat；
- 通用 `s=0` 与离散策略；
- `pdfFwd/pdfRev` ratio-walk MIS；
- delta/折射 eta/非互易校验；
- 光源功率 alias table。

### M3：介质与曲空间

- medium vertex、phase function、segment transmittance；
- 曲空间测地连接和 Jacobian；
- 对不支持的 shape/geometry 在配置阶段给出错误，而非运行期降级。

### M4：性能

- 每场景缓存 LightDistribution；
- arena/pool 化 path vertex；
- 批量 shadow rays；
- 连接深度与策略开关；
- benchmark：samples/s、有效连接率、shadow-ray 命中率、每策略方差。

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

当前实现应定义为“欧氏 3D 连续表面 BDPT”，而不是完整的 Veach BDPT。

在以下限制同时成立时，它对最大深度以内的路径积分基本无偏：

- 场景是三维 Euclidean 空间；
- 光源是有限、可按面积采样的双面面积光；
- 路径只包含 Lambert、rough conductor 等连续、互易的反射 BSDF；
- 不包含 delta、折射、介质、非互易散射或曲空间连接；
- 将 `MaxRayLevel` 视为积分目标的一部分。

若目标是无限深度的完整渲染方程，则固定最大深度仍会产生截断偏差。对 delta、介质、非互易或非 Euclidean 场景，当前 `bdpt` 配置会整体退回单向 path tracing，因此输出可能仍然有效，但不能作为 BDPT 能力或 BDPT 焦散质量的证据。

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

光源先按面积选择对象、再在对象表面均匀采样。若每个表面采样器确实返回 `1/Area(light)`，联合密度为

```text
p(light,x) = Area(light)/Area(total) × 1/Area(light)
           = 1/Area(total)
```

这在数学上无偏，但没有利用发光功率，会使面积很小而功率很高的光源产生较大方差。

### 9.3 不能扩展宣称无偏的部分

1. **固定深度截断。** 相机和光源子路径当前不执行 Russian roulette，只由 `MaxRayLevel` 截断。它对截断积分无偏，但不对无限路径空间严格无偏。
2. **策略族不完整。** 当前没有通用 `s=0`、`t=1` film splat、camera endpoint importance 和 delta 离散策略。对当前连续面积光支持域，这主要损失方差效率；对完整 BDPT 能力则属于缺失项。
3. **折射和介质未进入 BDPT。** 缺少 eta 修正、正反向离散/连续 PDF、medium vertex、phase function 和连接段 transmittance。
4. **三维假设未被类型契约完全表达。** 光源方向采样使用 3D cosine hemisphere，端点 PDF 使用 `|cos|/(2π)`，连接使用 `1/r²`。能力门禁目前只检查 Euclidean geometry；若将来 N 维 Euclidean shape 实现 `SurfaceSampler`，这些公式不能直接复用。
5. **发光模型隐含双面 Lambertian 方向分布。** 当前 emitter 的 `Emit` 都不依赖方向，因此与实现一致；若增加单面或方向性 emitter，必须同时增加方向分布、PDF 和能力声明。
6. **全场景回退影响可观察性。** 任意一个对象含介质边界、delta 或非互易 BSDF 时，整个场景静默退回 path tracing。运行结果不应被标记为“BDPT 已覆盖该传输类型”。

### 9.4 当前验证的解释

现有自动测试验证了：

- 面积光能够连接到相机子路径；
- 基础连接贡献有限、非负且符合一个简单解析值；
- 全局 MIS 权重形成单位分割；
- 不支持的传输类型触发能力门禁。

这些测试能够发现公式和策略归一化的局部回归，但尚不能证明统计无偏。还需要：

1. 解析直接光场景的多随机种子均值与置信区间；
2. Cornell box 中 BDPT 与高样本参考积分器的线性能量比较；
3. 按 `(s,t)` 分离策略后验证各策略期望及合并期望；
4. 多光源面积/功率差异测试；
5. RGB、hero wavelength、sampled wavelength 的能量一致性测试；
6. 明确断言实际采用的是 BDPT 还是 path fallback。

棱镜光谱示例包含 `specular_dielectric` 和 `medium_boundary`，因此当前会触发 path fallback。它可以验证光谱路径追踪和回退安全性，但不能验证 BDPT 折射链或双向焦散。

### 9.5 工程质量判断

- 数学骨架：良好；
- 受限域内正确性：较好；
- 完整 BDPT 覆盖度：较低；
- 方差效率：一般，光源按面积而不是按功率选择；
- 性能：全部顶点两两连接约为 `O(depth²)`，且当前没有 RR；
- 验证强度：已有单元级代数验证，缺少统计收敛验证。

生产化前应优先加入运行期积分器诊断、严格 3D 能力门禁、真实连续表面统计基准，再扩展 `t=1`、delta/eta 和介质路径。
