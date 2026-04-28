# `src-golang/controller` 场景交互 JSON 规则

本文档基于当前源码实现整理，目标是把 `D:\Algo\Projects\Ray\src-golang\controller` 实际消费的 JSON 结构、字段语义、默认行为、兼容性差异和已知限制说明清楚。

核心入口在以下文件：

- `D:\Algo\Projects\Ray\src-golang\controller\srcipt.go`
- `D:\Algo\Projects\Ray\src-golang\controller\parse_materials.go`
- `D:\Algo\Projects\Ray\src-golang\controller\parse_shape.go`

## 1. 总览

`controller` 当前把场景脚本反序列化到下面这个 Go 结构：

```go
type Script struct {
    Materials []map[string]interface{} `json:"materials"`
    Objects   []map[string]interface{} `json:"objects"`
    Cameras   []map[string]interface{} `json:"camera"`
}
```

这意味着：

- 顶层主要约定是 `materials`、`objects`、`camera`。
- 其中真正参与场景构建的只有 `materials` 和 `objects`。
- `camera` 字段虽然被定义在 `Script` 中，但当前 `LoadSceneFromScript` 并不会读取它。
- 渲染使用的相机目前由 `src-golang/handler.go` 中的 `BuildCamera(...)` 代码硬编码创建，而不是从 JSON 中加载。

## 2. 顶层 JSON 结构

推荐使用如下结构：

```json
{
  "materials": [],
  "objects": [],
  "camera": []
}
```

更完整的示例：

```json
{
  "materials": [
    {
      "id": "Light",
      "color": [10, 10, 10],
      "radiate": 1
    },
    {
      "id": "Wall",
      "color": [1, 1, 1],
      "reflectivity": 0,
      "refractivity": 0,
      "diffuse_loss": 0.7
    }
  ],
  "objects": [
    {
      "id": "room",
      "shape": "cuboid",
      "position": [0, 0, 0],
      "size": [5, 5, 5],
      "material_id": "Wall"
    },
    {
      "id": "lamp",
      "shape": "sphere",
      "position": [0, 0, 2],
      "r": 0.5,
      "material_id": "Light"
    }
  ],
  "camera": [
    {
      "position": [0, 0, 10]
    }
  ]
}
```

## 3. 顶层字段规则

| 字段 | 类型 | 是否被当前 controller 实际使用 | 说明 |
| --- | --- | --- | --- |
| `materials` | `array<object>` | 是 | 材质列表，加载时会先构建 `id -> material` 映射。 |
| `objects` | `array<object>` | 是 | 场景对象列表，每个对象通过 `material_id` 关联材质。 |
| `camera` | `array<object>` | 否 | 目前只定义了反序列化入口，未参与场景构建。 |

注意：

- 如果 JSON 语法错误，`ReadScriptFile(...)` 会返回 `nil`。
- 当前 `handler.LoadScript(...)` 没有对 `nil` 做额外保护，因此脚本文件应保证是合法 JSON。

## 4. 材质 `materials` 规则

每个材质元素本质上是一个任意键值对对象，`ParseMaterials(...)` 会按字段名做选择性解析。

### 4.1 必填字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 材质唯一标识，供对象通过 `material_id` 引用。 |
| `color` | `number[]` | 基础颜色，通常应为 3 个数字，对应 RGB。 |

如果 `id` 缺失或为空，最终会退化成空字符串键，容易覆盖其他材质，不建议这样使用。

### 4.2 可选字段

| 字段 | 类型 | 默认行为 | 说明 |
| --- | --- | --- | --- |
| `radiate` | `boolean` / `number` | `false` | 是否自发光。源码使用 `cast.ToBool(...)`，所以 `1/0` 也能被接受。 |
| `radiation_type` | `string` | `""` | 发光模式。当前实现支持普通发光和 `"directional light source"`。 |
| `reflectivity` | `number` | `0` | 反射概率。 |
| `refractivity` | `number` | `0` | 折射概率。 |
| `refractive_index` | `number` / `number[]` | `nil` | 折射率，支持单值或数组。 |
| `diffuse_loss` | `number` | `1.0` | 漫反射损耗系数。 |
| `reflect_loss` | `number` | `1.0` | 反射损耗系数。 |
| `refract_loss` | `number` | `1.0` | 折射损耗系数。 |
| `color_func` | `string` | `nil` | 颜色函数名称，从 `optics.ColorFuncMap` 中查找。 |

### 4.3 `refractive_index` 的实际含义

`refractive_index` 有两种写法：

1. 单个数字

```json
{
  "refractive_index": 1.5
}
```

此时表示常量折射率。

2. 数组

```json
{
  "refractive_index": [1.0, 50000, 0]
}
```

此时源码会把它当成长度为 3 的向量处理，并在光线传播时通过 `utils.CauchyDispersion(...)` 计算色散折射率。

建议：

- 如果只是普通透明材质，优先使用单个数字。
- 如果需要色散效果，再使用 3 个数字的数组。

### 4.4 当前可用的 `color_func`

当前 `src-golang/model/optics/color_func_lib.go` 里只有一个注册值：

```json
{
  "color_func": "color_func_1"
}
```

它会根据法线方向返回不同颜色。

如果写入未注册的名称：

- 不会报错。
- 但查表结果为 `nil`，最终仍然回退为普通 `color`。

## 5. 对象 `objects` 规则

每个对象都至少需要：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 对象标识，当前加载逻辑不会直接使用，但建议保留用于排查和编辑器展示。 |
| `shape` | `string` | 形状类型，决定后续读取哪些字段。 |
| `material_id` | `string` | 关联到某个 `materials[i].id`。 |

### 5.1 `material_id` 的行为

`LoadSceneFromScript(...)` 会先查材质映射：

- 如果 `material_id` 找不到对应材质，对象会被直接跳过。
- 当前不会报错，也不会给出警告。

因此文档层面建议把 `material_id` 视为必填且必须有效。

### 5.2 当前支持的 `shape` 值

`ParseShape(...)` 当前识别以下字符串：

- `cuboid`
- `sphere`
- `triangle`
- `plane`
- `quadratic equation`
- `four-order equation`
- `stl`

其中需要特别注意：

- `plane` 虽然出现在分支里，但当前没有实际构造逻辑，最终会返回空对象。
- `stl` 在 controller 中是支持的，但 `scene-editor` 当前类型定义和面板 schema 里没有暴露这个类型。

## 6. 各 `shape` 的字段规则

### 6.1 `cuboid`

有两种写法。

方式 A：使用中心点和尺寸。

```json
{
  "shape": "cuboid",
  "position": [0, 0, 0],
  "size": [5, 5, 5]
}
```

解释：

- `position` 是中心点。
- `size` 是整体尺寸，不是半尺寸。
- 源码会自动计算：
  - `pmin = position - size * 0.5`
  - `pmax = position + size * 0.5`

方式 B：直接使用边界框。

```json
{
  "shape": "cuboid",
  "pmin": [-2.5, -2.5, -2.5],
  "pmax": [2.5, 2.5, 2.5]
}
```

规则：

- 优先判断 `position` 是否存在。
- 如果存在 `position`，就走 `position + size` 模式。
- 否则如果存在 `pmax`，就按 `pmin + pmax` 模式构造。
- 如果两套字段都不完整，当前对象会被忽略。

可选字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `engraving_func` | `string` | 雕刻/裁剪函数名。 |

注意：

- parser 允许 `cuboid` 写 `engraving_func`。
- 但当前注册的唯一函数 `"sphere1"` 是按球体参数读取的，不适用于 `cuboid`。
- 如果给 `cuboid` 配置 `"sphere1"`，运行时存在 panic 风险。

### 6.2 `sphere`

```json
{
  "shape": "sphere",
  "position": [1.2, 0, 0],
  "r": 0.9
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `position` | `number[]` | 球心。 |
| `r` | `number` | 半径。 |
| `engraving_func` | `string` | 可选雕刻函数名。 |

当前可用的 `engraving_func`：

```json
{
  "engraving_func": "sphere1"
}
```

这个函数会基于球面位置生成螺旋状挖空效果。

### 6.3 `triangle`

```json
{
  "shape": "triangle",
  "p1": [0, 0, 0],
  "p2": [1, 0, 0],
  "p3": [0, 1, 0]
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `p1` | `number[]` | 第一个顶点。 |
| `p2` | `number[]` | 第二个顶点。 |
| `p3` | `number[]` | 第三个顶点。 |

注意：

- 三角形法线方向取决于顶点顺序。
- 双面/背面行为会受到交点计算逻辑影响，因此顶点绕序最好保持一致。

### 6.4 `plane`

当前 `ParseShape(...)` 虽然保留了：

```json
{
  "shape": "plane"
}
```

但没有真正创建 `plane` 对象。

实际效果：

- 该对象最终不会生成任何几何体。
- 可以认为当前版本的 `plane` 尚未实现。

### 6.5 `quadratic equation`

```json
{
  "shape": "quadratic equation",
  "a": [
    -20, 0, 0,
    0, -10, 0,
    0, 0, -20
  ],
  "b": [40, 0, 0],
  "c": -10
}
```

其数学形式为：

```text
f(x) = x^T A x + b^T x + c
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `a` | `number[]` | 3x3 矩阵，按行展开，长度应为 9。 |
| `b` | `number[]` | 线性项向量，通常为 3 个数字。 |
| `c` | `number` | 常数项。 |

注意：

- 当前实现固定使用 `mat.NewDense(3, 3, ...)`，因此 `a` 最好严格写 9 个数字。
- `b` 也应与维度匹配，通常写 3 个数字。
- `scene-editor` 中会额外带一个 `position` 作为预览锚点，但 controller 不会读取这个字段。

### 6.6 `four-order equation`

```json
{
  "shape": "four-order equation",
  "a": [/* 扁平化四阶张量系数 */]
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `a` | `number[]` | 四阶张量的扁平数组。 |

实现细节：

- 源码调用 `math_lib.NewTensorFromSlice(A, []int{4, 4, 4, 4})`。
- 这意味着最稳妥的写法是提供 `4 * 4 * 4 * 4 = 256` 个系数。
- 索引语义在源码注释中约定为：`0 -> 常量 1`，`1 -> x`，`2 -> y`，`3 -> z`。

因此单项式的构造不是“按次数列表直接填”，而是“按四阶张量展开后填系数”。

同样地：

- `scene-editor` 里可能带 `position` 预览字段。
- controller 不会消费这个字段。

### 6.7 `stl`

```json
{
  "shape": "stl",
  "file": "D:/path/to/model.stl",
  "position": [0, 0, 0],
  "z_dir": [0, 0, 1],
  "x_dir": [1, 0, 0],
  "scale": [1, 1, 1]
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `file` | `string` | STL 文件路径。 |
| `position` | `number[]` | 平移位置。 |
| `z_dir` | `number[]` | 目标局部 Z 轴方向。 |
| `x_dir` | `number[]` | 目标局部 X 轴方向。 |
| `scale` | `number[]` | 各轴缩放。 |

行为说明：

- controller 会读取 STL 文件并转换为三角形列表。
- 同时支持 ASCII STL 和二进制 STL。
- 结果不是一个单独的复合对象，而是被展开成多个 `triangle` 形状加入场景。

约束和风险：

- `file` 必须是当前运行环境下可访问的真实路径。
- `z_dir` 和 `x_dir` 应为非零且尽量正交的向量，否则变换矩阵可能异常。
- STL 读取失败时当前实现会直接 `panic`，不是优雅报错。

## 7. 数值与类型约定

虽然源码把 JSON 先反序列化到 `map[string]interface{}`，再借助 `cast` 做转换，但为了稳定起见，建议遵守以下规范。

### 7.1 向量

向量字段统一建议写成数字数组：

```json
[x, y, z]
```

常见字段包括：

- `color`
- `position`
- `size`
- `pmin`
- `pmax`
- `p1`
- `p2`
- `p3`
- `b`
- `z_dir`
- `x_dir`
- `scale`

### 7.2 布尔值

`radiate` 虽然示例里常写成 `1`，但从语义上它是布尔开关。

当前实现可接受：

- `true` / `false`
- `1` / `0`

推荐优先保持和现有示例一致，或者在文档约定里明确团队统一写法。

### 7.3 未知字段

当前 parser 只读取自己关心的字段：

- 多余字段不会报错。
- 但也不会生效。

这对编辑器扩展字段比较友好，但也意味着字段写错名字时不容易被及时发现。

## 8. 当前加载流程

加载顺序如下：

1. `ReadScriptFile(filepath)` 读取文件并执行 `json.Unmarshal(...)`
2. `LoadSceneFromScript(script, scene)` 先调用 `ParseMaterials(script)`
3. 遍历 `script.Objects`
4. 用 `material_id` 查找材质
5. 调用 `ParseShape(item)` 生成一个或多个 `shape.Shape`
6. 把每个 shape 包装成 `object.Object` 后加入 `scene.ObjectTree`
7. 调用 `scene.ObjectTree.Build()`

其中一个对象可能对应多个实际图元：

- 普通 `cuboid` / `sphere` / `triangle` / `quadratic equation` / `four-order equation` 通常只生成 1 个 shape。
- `stl` 会生成很多个 `triangle`。

## 9. 与 `scene-editor` 的差异

这是当前最值得特别记录的一组兼容性问题。

### 9.1 `camera` vs `cameras`

`scene-editor` 的 `SceneDocument` 定义是：

```ts
{
  materials: SceneMaterial[];
  objects: SceneObject[];
  cameras: SceneCamera[];
}
```

但 `controller.Script` 里定义的是：

```go
json:"camera"
```

并且当前 controller 根本没有消费相机配置。

结论：

- 编辑器导出的 `cameras` 字段当前不会被 controller 使用。
- 就算改成 `camera`，当前 controller 也仍然不会真正拿它建相机。

### 9.2 `scene-editor` 暴露了 `plane`，但后端未实现

编辑器允许创建 `plane`，但 controller 解析后不会生成几何体。

### 9.3 `scene-editor` 未暴露 `stl`

controller 支持 `shape: "stl"`，但编辑器的 `ShapeType` 和 schema 里暂时没有这个类型。

### 9.4 编辑器中的预览字段不一定被后端消费

例如：

- `quadratic equation` 的 `position`
- `four-order equation` 的 `position`

这些字段在编辑器里用于可视化代理或锚点，但 controller 当前不会读取。

## 10. 推荐写法

如果目标是和当前 `src-golang/controller` 保持稳定兼容，推荐遵循下面这些原则：

- 顶层只依赖 `materials` 和 `objects`，不要指望相机 JSON 生效。
- 每个材质必须有唯一 `id` 和合法 `color`。
- 每个对象必须有有效的 `material_id`。
- `cuboid` 优先使用 `position + size` 写法，表达更清晰。
- `quadratic equation` 的 `a` 固定写 9 个数字。
- `four-order equation` 的 `a` 按 256 个系数准备。
- `plane` 先不要用于正式场景。
- `stl` 使用绝对路径或稳定相对路径前，先确认运行目录和文件可达性。
- `engraving_func` 当前仅建议用于 `sphere`，并使用 `"sphere1"`。
- `color_func` 当前仅有 `"color_func_1"` 可选。

## 11. 一个兼容当前 controller 的完整示例

```json
{
  "materials": [
    {
      "id": "LightSource",
      "color": [10.0, 10.0, 10.0],
      "radiate": 1
    },
    {
      "id": "WorldBoundary",
      "color": [1.0, 1.0, 1.0],
      "reflectivity": 0.0,
      "refractivity": 0.0,
      "diffuse_loss": 0.7
    },
    {
      "id": "mirror",
      "color": [1.0, 1.0, 1.0],
      "reflectivity": 1.0,
      "refractivity": 0.0
    }
  ],
  "objects": [
    {
      "id": "WorldBox",
      "shape": "cuboid",
      "position": [0, 0, 0],
      "size": [5, 5, 5],
      "material_id": "WorldBoundary"
    },
    {
      "id": "MainLight",
      "shape": "sphere",
      "position": [0, 0, 2],
      "r": 0.6,
      "material_id": "LightSource"
    },
    {
      "id": "MirrorBall",
      "shape": "sphere",
      "position": [1.2, 0, 0],
      "r": 0.9,
      "material_id": "mirror"
    }
  ],
  "camera": []
}
```

## 12. 结论

当前 `src-golang/controller` 的 JSON 规则更接近“弱 schema + 按字段名解析”的风格：

- 结构比较灵活。
- 容错表面上较宽松。
- 但很多字段写错时不会显式报错，而是静默跳过或运行时失败。

如果后续要把这个协议稳定成对外文档，优先建议补三件事：

1. 统一 `camera` / `cameras` 命名，并真正接入相机加载。
2. 明确 `plane`、`stl`、`engraving_func` 的正式支持范围。
3. 给关键字段缺失和非法值增加显式错误信息，而不是静默跳过。
