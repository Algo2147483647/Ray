# 非欧光线渲染设计 (Non-Euclidean Rendering)

**Date**: 2026-05-31
**Scope**: 在 Go 渲染引擎中加入对双曲空间 (H³) 和球面空间 (S³) 的光线追踪支持。
**Status**: Draft, awaiting user review.

## 1. 目标 & 非目标

### 目标

- 让 `engine/` 支持可切换的全局几何：`euclidean` (现状) / `klein` (H³) / `spherical` (S³)。
- 渲染管线（相机 → BVH → 相交 → BSDF → 介质）整体由 Scene 的几何设置驱动；K=0 (Euclidean) 时**字节级等价**当前实现。
- 验收：能渲染 (a) 双曲棋盘 (b) S³ 中的 Hopf 环（含 S³ 环绕效果）。

### 非目标

- 不实现 Poincaré / Hyperboloid / 任意黎曼度规 / 数值积分测地线；这些预留为未来同接口下的扩展实现。
- 不实现 N 维 (N>3) 的双曲/球面；嵌入维度按 Klein=3、Spherical=4 固定。
- BSDF 模型不重做；现有 BSDF 在切空间上原样复用。

## 2. 关键设计决策（已与用户确认）

| 决策点       | 取值                                                    |
| ------------ | ------------------------------------------------------- |
| 几何范围     | 可切换的全局空间（Scene 级）                            |
| 几何模型     | 双曲: Beltrami-Klein；球面: S³ in R⁴                    |
| 原语坐标语义 | 模型坐标（用户给的 center/r 就是 Klein/S³ 嵌入坐标）    |
| 局部物理     | 在切空间上原样复用 BSDF/采样/介质                       |
| 相机         | 原生双曲/球面相机（新增两个类）                         |
| 维度         | 仅 3D 主场景（嵌入维 3 或 4）                           |
| S³ 环绕      | 默认继续传播，配 `max_arc` 上限（默认 2π）              |
| 抽象位置     | 新建 `Geometry` 接口；现有代码就地接入                  |
| Klein 边界   | 不特殊处理；介质 β 天然趋零；hit Klein 球边界判为未命中 |
| 路径终止     | 现有 level-based RR 不动；另加 `max_arc` 弧长上限       |

## 3. 架构 & 模块边界

```
engine/
  model/
    geometry/                      ← 新增
      geometry.go                  Geometry 接口 + Euclidean 单例
      euclidean.go
      klein.go                     H³ via Beltrami-Klein 模型 (R³ 单位球)
      spherical.go                 S³ via R⁴ 单位向量
      geometry_test.go
    optics/ray.go                  Ray 增 Geometry 字段（nil → Euclidean）
    camera/
      camera_hyperbolic.go         ← 新增
      camera_spherical.go          ← 新增
    shape/                         不动（仍按嵌入欧氏射线相交）
    object/                        不动（BVH 复用）
  ray_tracing/
    trace_ray.go                   distance→弧长；S³ 环绕；约 5 个调用点改动
  sceneio/
    scheme/factory                 解析 "geometry" 字段，构造对应 Geometry 单例
examples/scenes/
  hyperbolic_chessboard.json       ← 新增（验收场景）
  spherical_hopf.json              ← 新增（验收场景）
```

## 4. `Geometry` 接口

```
// engine/model/geometry/geometry.go

type Geometry interface {
    Name() string
    Dimension() int                                          // 嵌入维度：Klein=3, Spherical=4

    // 把任意向量投影到 T_p（位置 p 处的切空间）
    ProjectTangent(p, v *mat.VecDense, out *mat.VecDense) *mat.VecDense

    // 切空间内积 ⟨u,v⟩_p
    InnerProduct(p, u, v *mat.VecDense) float64

    // 从嵌入射线参数 t 计算测地弧长
    // 不变量：调用方刚刚用 (p, dir) 在嵌入欧氏意义下求交得到 t
    ArcLengthFromEmbedT(p, dir *mat.VecDense, tEuclid float64) float64

    // 测地线参数化：γ(t) = Exp_p(t·v)
    Exp(p, v *mat.VecDense, t float64, out *mat.VecDense) *mat.VecDense

    // 嵌入射线相交参数化所需信息：
    // 返回 BVH/Shape 可以使用的 (eo, ed) 以及到"自然终点"的最大欧氏 t（Klein 球边界 / S³ 半圆等）
    EmbeddedRay(p, dir *mat.VecDense) (eo, ed *mat.VecDense, tMaxEmbed float64)

    // S³ 专用：当一段无相交时，沿测地线推进 arcAdvance 并把方向 parallel-transport
    // 到新位置上。Euclidean / Klein 返回 (_, _, false)。
    // 通常 arcAdvance 取 g.EmbeddedRay 返回的 tMaxEmbed（S³ 为 π）。
    WrapBeyond(p, dir *mat.VecDense, arcAdvance float64) (newP, newD *mat.VecDense, ok bool)
}
```

**实现要点：**

| 方法                         | Euclidean  | Klein (H³)                   | Spherical (S³)                   |
| ---------------------------- | ---------- | ---------------------------- | -------------------------------- |
| `Dimension()`                | 3          | 3                            | 4                                |
| `ProjectTangent(p,v)`        | v          | v                            | v − ⟨v,p⟩p                       |
| `InnerProduct(p,u,v)`        | u·v        | Klein 度规公式 (依赖 \|p\|²) | u·v                              |
| `ArcLengthFromEmbedT(p,d,t)` | t·\|d\|    | atanh-based 闭式             | acos-based 闭式                  |
| `Exp(p,v,t)`                 | p + t·v    | Klein 测地参数化             | cos(t)·p + sin(t)·v̂              |
| `EmbeddedRay(p,d)`           | (p, d, +∞) | (p, d, t_到 Klein 球边界)    | (p, d, π)                        |
| `WrapBeyond`                 | nil, false | nil, false                   | (Exp(p,d,arcUsed), 平移 d, true) |

Klein 度规、双曲距离、双曲测地参数化的闭式公式见 Cannon & Floyd, *Hyperbolic Geometry*；实现处给注释引用。

**单例化**：`Euclidean()` / `Klein()` / `Spherical()` 返回 package 级单例（无状态）。`Ray.Geometry == nil` 时 `ray.G()` 帮助方法返回 `Euclidean()`，保证向后兼容。

## 5. 数据流（一次 path tracing）

```
1. Camera.SpawnRay(pixel):
   - origin   = 相机位置（嵌入坐标，已在 S³ / Klein 球内）
   - tangent  = 在 T_origin 内构造的像素方向（按 FOV / film 坐标）
   - direction = ProjectTangent(origin, embed(tangent)) 并按嵌入欧氏归一
   - ray.Geometry = scene.Geometry

2. TraceRay(ray, level):
   a. hit, ok = objTree.GetSurfaceHit(ray.O, ray.D)  ← BVH 原样
                hit.Distance 是嵌入欧氏 t
   b. (Klein) 若 hit 表示击中 Klein 球边界 → 视为未命中，取背景，结束
   c. arcLen = g.ArcLengthFromEmbedT(ray.O, ray.D, hit.Distance)
   d. media.ApplyAbsorption(ray, arcLen)                 ← 用弧长
   e. frame = NewFrameInGeometry(g, hit.Point, hit.ShadingNormal)
   f. 处理 emission；BSDF.Sample → wi_local              ← 不动
   g. wi_tangent = frame.LocalToWorld(wi_local)
   h. wi = g.ProjectTangent(hit.Point, wi_tangent) 并按 g.InnerProduct 归一
   i. ray.O = hit.Point;  ray.D = wi
   j. 累计 arcLen 到 ray.ArcTraveled；超 max_arc → 终止
   k. recurse

3. 未命中：
   - Euclidean / Klein: 取环境光，结束
   - Spherical: 若 ray.ArcTraveled < max_arc：
       (newO, newD, _) = g.WrapBeyond(ray.O, ray.D, π)
       ray.O, ray.D = newO, newD；arcUsed += π；继续 (a)
     若 ArcTraveled ≥ max_arc：取背景，结束
```

### 关键不变量

- `Ray.Direction` 在嵌入域里是**测地线方向**；Klein 下恰好是欧氏直线方向 → BVH/Shape 无需感知几何。
- `hit.Distance`（欧氏 t）在 `TraceRay` 内**立即**翻译为 `arcLen`；下游全部用 `arcLen`，避免语义混淆。
- `Frame` 永远在 `T_p` 内，BSDF 看到的 (wi, wo) 都是局部欧氏单位向量；cos/PDF 不变。

### 已知风险

- Klein 在边界附近 `arcLen → ∞`：需在 `ArcLengthFromEmbedT` 内 clamp 到 `MaxArc * 4` 之类保险值防 Inf 污染；对介质/RR 而言已经"看不见"，物理正确。
- S³ wrap 后 ray 方向需要 parallel transport；在 Klein/Euclidean 不存在此需求。`WrapBeyond` 内做闭式 transport（不是 generic ParallelTransport）。

## 6. 组件细节

### 6.1 `Ray` 改造

```
// engine/model/optics/ray.go
type Ray struct {
    // ...原字段...
    Geometry     geometry.Geometry  // nil ⇒ Euclidean
    ArcTraveled  float64            // S³ 用；其他几何可忽略
}
func (r *Ray) G() geometry.Geometry { /* nil-safe */ }
```

### 6.2 `Frame` 改造

新增 `NewFrameFromNormalInGeometry(g, p, n)`：

- Euclidean / Klein 3D：保持现有 Cross2 路径。
- Spherical：先把 `n` 投到 `T_p`，在 `T_p` 这 3 维子空间里做 Gram-Schmidt 得到 3 个 tangent；存为长度 4 的向量但秩 3。

`WorldToLocal` / `LocalToWorld` 实现不变（dot/sum 已经自然适配）。

### 6.3 `Camera`

`HyperbolicCamera` / `SphericalCamera`：

- 复用 `Camera3D` 的 film / pixel 采样路径（embed via composition）。
- 仅 `SpawnRay` 不同：用相机位置 + 朝向构造 T_p 内的像素方向，再 embed 为嵌入坐标。
- Scene JSON `"camera": { "type": "hyperbolic", ... }` 走 sceneio factory 选型。

### 6.4 `trace_ray.go` 改动点（清单）

| 位置                        | 改动                                                         |
| --------------------------- | ------------------------------------------------------------ |
| `TraceRay` 入口             | 增加 `g := ray.G()`                                          |
| hit 后                      | `arcLen := g.ArcLengthFromEmbedT(...)`；把现有 `hit.Distance` 替换为 `arcLen` 传给介质 |
| `prepareSurfaceInteraction` | Frame 构造改 `NewFrameFromNormalInGeometry(g, ...)`          |
| `applySurfaceSample`        | `sample.Wi` → world tangent → `ProjectTangent` + normalize   |
| 未命中分支                  | 按几何分支：Spherical wrap + 弧长累计                        |

预计实际改动 ~80 行。

### 6.5 `sceneio` schema 增量

```
{
  "geometry": { "type": "klein" },        // or "spherical" / "euclidean"（默认）
  "max_arc": 12.566,                      // 可选，默认：spherical=2π, klein=∞
  "camera": { "type": "hyperbolic", ... },
  "objects": [ ... ]
}
```

工厂端在 Scene 构造时 attach Geometry 单例；TraceHandler 派发 Ray 时填上 `ray.Geometry`。

### 6.6 验收场景

```
examples/scenes/hyperbolic_chessboard.json
```

- Klein 球内 z≈0 切片放一张棋盘，Lambert + Mirror 混合
- 期望：远处方块在视觉上指数级缩小，相互弯曲——经典 H² 棋盘效果

```
examples/scenes/spherical_hopf.json
```

- S³ 中两个互锁的 Hopf 圆环（半径 1/√2 的 Clifford 环面上）
- max_arc=2π
- 期望：相机能看到自己被反足点放大反射的镜像

## 7. 测试策略

### 单元测试 `geometry_test.go`

- Euclidean 各方法 = identity / 欧氏；
- Klein：`Distance(0,p) = atanh(|p|)`；`Exp(0,v,t)` 与 `Log` 互为逆；`Exp(p,v,0) = p`；
- Spherical：`Distance(p,-p) = π`；`Exp(p,v,2π) ≈ p`；`Distance(p,p) = 0`；wrap 后方向与 transported 方向一致。

### 回归测试

现有所有渲染场景在默认 (Euclidean) 下输出位等价或哈希不变。`engine/ray_tracing/trace_*_test.go` 跑通。

### 集成场景

- `hyperbolic_chessboard.json` 渲染产物入 `docs/assets/`，作为视觉回归基线。
- `spherical_hopf.json` 同上。

### 性能预算

Euclidean 路径增量 < 2%（Geometry 接口走 Euclidean 单例，方法体足够小可被编译器 inline）。CI 增一个 bench gate。

## 8. 演进 / 非本次实现

以下接口下的扩展项已**预留 hook**，本次不实现：

- Poincaré 球模型（同 H³，不同投影；接口不动，新加 `poincare.go`）；
- Hyperboloid 模型（Minkowski 嵌入，复用 N-d 框架）；
- 一般可微度规 + RK4 数值测地线（黑洞、引力透镜场景）——新增 `riemannian.go` 实现 `Geometry`。
- Scene-editor 端的实时游走（更大项目，本次不含）。

## 9. 风险摘要

| 风险                                  | 处理                                                         |
| ------------------------------------- | ------------------------------------------------------------ |
| K=0 路径 regress                      | Euclidean 单例方法体最小化 + 全部现有测试守门                |
| Klein 边界 NaN/Inf                    | `ArcLengthFromEmbedT` clamp；边界击中 → 当作未命中           |
| S³ wrap 后方向漂移                    | `WrapBeyond` 用闭式 transport（不依赖 generic 实现）         |
| Frame 在 S³ 退化（n 与 p 共线极端）   | `NewFrameFromNormalInGeometry` 检查并 fallback               |
| BSDF 采样在切空间度规与嵌入度规不一致 | 文档化"BSDF 始终见到欧氏单位向量"；归一在 ProjectTangent 之后用 g.InnerProduct |

**下一步**：经用户审阅本 spec 后，进入 writing-plans 阶段，拆出可在子会话中执行的实现 plan。