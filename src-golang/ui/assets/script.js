// 全局变量
let sceneData = null;
let scene = null;
let camera = null;
let renderer = null;
let controls = null;
const objects = [];

// DOM元素
const jsonInput = document.getElementById('json-input');
const confirmBtn = document.getElementById('confirm-btn');
const resetBtn = document.getElementById('reset-btn');
const objectsTable = document.getElementById('objects-table');
const cuboidCount = document.getElementById('cuboid-count');
const sphereCount = document.getElementById('sphere-count');
const totalCount = document.getElementById('total-count');
const lightCount = document.getElementById('light-count');

// 初始化Three.js场景
function initScene() {
    // 创建场景
    scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0a0a1a);

    // 创建相机
    camera = new THREE.PerspectiveCamera(75,
        document.getElementById('scene-container').clientWidth /
        document.getElementById('scene-container').clientHeight,
        0.1, 10000);
    camera.position.set(0, 0, 3000);

    // 创建渲染器
    renderer = new THREE.WebGLRenderer({
        canvas: document.getElementById('scene-canvas'),
        antialias: true
    });
    renderer.setSize(
        document.getElementById('scene-container').clientWidth,
        document.getElementById('scene-container').clientHeight
    );

    // 添加轨道控制器
    controls = new THREE.OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.05;

    // 添加环境光
    const ambientLight = new THREE.AmbientLight(0x404040, 1.5);
    scene.add(ambientLight);

    // 添加方向光
    const directionalLight = new THREE.DirectionalLight(0xffffff, 1);
    directionalLight.position.set(1, 1, 1);
    scene.add(directionalLight);

    // 添加坐标轴辅助
    const axesHelper = new THREE.AxesHelper(1000);
    scene.add(axesHelper);

    // 开始动画循环
    animate();
}

// 动画循环
function animate() {
    requestAnimationFrame(animate);
    controls.update();
    renderer.render(scene, camera);
}

// 解析JSON并渲染场景
function parseAndRender() {
    try {
        // 解析JSON
        sceneData = JSON.parse(jsonInput.value);

        // 清除之前的物体
        objects.forEach(obj => scene.remove(obj));
        objects.length = 0;

        // 创建几何体
        let cuboids = 0;
        let spheres = 0;
        let lights = 0;

        sceneData.objects.forEach(obj => {
            let geometry, material, mesh;

            // 根据材质类型设置颜色
            let color = 0xffffff;
            const materialInfo = sceneData.materials.find(m => m.id === obj.material_id);
            if (materialInfo) {
                // 将颜色值从0-1范围转换为0-255范围，然后组合成十六进制颜色
                if (materialInfo.color) {
                    const r = Math.min(255, Math.floor(materialInfo.color[0] * 255));
                    const g = Math.min(255, Math.floor(materialInfo.color[1] * 255));
                    const b = Math.min(255, Math.floor(materialInfo.color[2] * 255));
                    color = (r << 16) | (g << 8) | b;
                }
                
                if (materialInfo.radiate) {
                    lights++;
                }
            }

            // 创建材质
            material = new THREE.MeshPhongMaterial({
                color: color,
                transparent: obj.material_id === 'Glass',
                opacity: obj.material_id === 'Glass' ? 0.7 : 1.0,
                wireframe: obj.id === 'WorldBox'
            });

            // 创建几何体
            if (obj.shape === 'cuboid') {
                geometry = new THREE.BoxGeometry(obj.size[0], obj.size[1], obj.size[2]);
                cuboids++;
            } else if (obj.shape === 'sphere') {
                const radius = obj.r || obj.radius || 100; // 支持r和radius两种字段
                geometry = new THREE.SphereGeometry(radius, 32, 32);
                spheres++;
            } else if (obj.shape === 'triangle') {
                // 创建三角形几何体
                const p1 = new THREE.Vector3(...obj.p1);
                const p2 = new THREE.Vector3(...obj.p2);
                const p3 = new THREE.Vector3(...obj.p3);
                geometry = new THREE.BufferGeometry();
                const vertices = new Float32Array([
                    p1.x, p1.y, p1.z,
                    p2.x, p2.y, p2.z,
                    p3.x, p3.y, p3.z
                ]);
                geometry.setAttribute('position', new THREE.BufferAttribute(vertices, 3));
                geometry.computeVertexNormals();
                // 三角形不计入统计
            }

            // 创建网格
            if (geometry) {
                mesh = new THREE.Mesh(geometry, material);
                mesh.position.set(obj.position ? obj.position[0] : 0, 
                                  obj.position ? obj.position[1] : 0, 
                                  obj.position ? obj.position[2] : 0);
                scene.add(mesh);
                objects.push(mesh);
            }
        });

        // 更新统计 (只统计立方体和球体)
        cuboidCount.textContent = cuboids;
        sphereCount.textContent = spheres;
        totalCount.textContent = cuboids + spheres;
        lightCount.textContent = lights;

        // 生成参数表格
        generateObjectsTable();

    } catch (e) {
        alert('JSON解析错误: ' + e.message);
        console.error(e);
    }
}

// 生成几何体参数表格
function generateObjectsTable() {
    if (!sceneData || !sceneData.objects) return;
    
    let tableHTML = `
        <table>
            <thead>
                <tr>
                <th>ID</th>
                <th>Shape</th>
                <th>Position (X, Y, Z)</th>
                <th>Size / Radius</th>
                <th>Material ID</th>
                <th>Action</th>
            </tr>
        </thead>
        <tbody>
        `;

    sceneData.objects.forEach((obj, index) => {
        tableHTML += `
            <tr>
                <td>${obj.id || ''}</td>
                <td>
                    <span class="shape-icon">${obj.shape === 'cuboid' ? '◼' : obj.shape === 'sphere' ? '●' : '△'}</span>
                    ${obj.shape}
                </td>
                <td>
        `;
        
        // 显示位置信息（如果存在）
        if (obj.position) {
            tableHTML += `
                    <input type="number" id="pos-x-${index}" value="${obj.position[0]}" min="-2000" max="2000">
                    <input type="number" id="pos-y-${index}" value="${obj.position[1]}" min="-2000" max="2000">
                    <input type="number" id="pos-z-${index}" value="${obj.position[2]}" min="-2000" max="2000">
                `;
        } else if (obj.p1 && obj.p2 && obj.p3) {
            // 三角形的三个点
            tableHTML += `
                P1: [${obj.p1.join(', ')}]<br/>
                P2: [${obj.p2.join(', ')}]<br/>
                P3: [${obj.p3.join(', ')}]
            `;
        }

        tableHTML += `
                </td>
                <td>
            `;

        if (obj.shape === 'cuboid') {
            tableHTML += `
                    <input type="number" id="size-w-${index}" value="${obj.size[0]}" min="1" max="2000">
                    <input type="number" id="size-h-${index}" value="${obj.size[1]}" min="1" max="2000">
                    <input type="number" id="size-d-${index}" value="${obj.size[2]}" min="1" max="2000">
                `;
        } else if (obj.shape === 'sphere') {
            const radius = obj.r || obj.radius || 100;
            tableHTML += `
                    <input type="number" id="radius-${index}" value="${radius}" min="1" max="1000">
                `;
        } else {
            tableHTML += `N/A`;
        }

        tableHTML += `
                </td>
                <td>${obj.material_id || obj.material || ''}</td>
                <td>
                    <button class="update-btn" data-index="${index}">更新</button>
                </td>
            </tr>
            `;
    });

    tableHTML += `
            </tbody>
        </table>
        `;

    objectsTable.innerHTML = tableHTML;

    // 添加更新按钮事件监听
    document.querySelectorAll('.update-btn').forEach(btn => {
        btn.addEventListener('click', function() {
            const index = this.getAttribute('data-index');
            updateObject(index);
        });
    });
}

// 更新几何体参数
function updateObject(index) {
    // 获取输入值
    const posX = parseFloat(document.getElementById(`pos-x-${index}`).value) || 0;
    const posY = parseFloat(document.getElementById(`pos-y-${index}`).value) || 0;
    const posZ = parseFloat(document.getElementById(`pos-z-${index}`).value) || 0;

    // 更新位置
    sceneData.objects[index].position = [posX, posY, posZ];

    // 更新尺寸
    if (sceneData.objects[index].shape === 'cuboid') {
        const sizeW = parseFloat(document.getElementById(`size-w-${index}`).value) || 100;
        const sizeH = parseFloat(document.getElementById(`size-h-${index}`).value) || 100;
        const sizeD = parseFloat(document.getElementById(`size-d-${index}`).value) || 100;
        sceneData.objects[index].size = [sizeW, sizeH, sizeD];
    } else if (sceneData.objects[index].shape === 'sphere') {
        const radius = parseFloat(document.getElementById(`radius-${index}`).value) || 100;
        // 同时更新r和radius字段以保持兼容性
        sceneData.objects[index].r = radius;
        if (sceneData.objects[index].radius !== undefined) {
            sceneData.objects[index].radius = radius;
        }
    }

    // 更新JSON输入框
    jsonInput.value = JSON.stringify(sceneData, null, 2);

    // 重新渲染场景
    parseAndRender();
}

// 重置配置
function resetConfig() {
    jsonInput.value = `{
    "materials": [
        {
            "id": "Paper",
            "color": [1, 1, 1],
            "diffuse_loss": 1,
            "reflect_loss": 0,
            "refract_loss": 0,
            "refractivity":0
        },
        {
            "id": "Glass",
            "color": [1, 1, 1],
            "diffuse_loss": 0,
            "reflect_loss": 1,
            "refract_loss": 0,
            "refractivity":1.7
        },
        {
            "id": "Metal",
            "color": [1, 1, 0],
            "diffuse_loss": 0.5,
            "reflect_loss": 0.5,
            "refract_loss": 0,
            "refractivity": 0
        },
        {
            "id": "Light",
            "color": [10, 10, 10],
            "radiate": 1
        }
    ],
    "objects": [
        {
            "id": "box1",
            "shape": "cuboid",
            "position": [0, 0, 0],
            "size": [2000, 2000, 2000],
            "material_id": "Paper"
        },
        {
            "id": "glass_panel1",
            "shape": "cuboid",
            "position": [850, 1320, 0],
            "size": [1150, 1350, 300],
            "material_id": "Glass"
        },
        {
            "id": "glass_panel2",
            "shape": "cuboid",
            "position": [850, 900, 0],
            "size": [1150, 930, 300],
            "material_id": "Glass"
        },
        {
            "id": "glass_panel3",
            "shape": "cuboid",
            "position": [850, 900, 0],
            "size": [1150, 930, 300],
            "material_id": "Glass"
        },
        {
            "id": "light_source",
            "shape": "sphere",
            "position": [1000, 1000, 1600],
            "r": 400,
            "material_id": "Light"
        }
    ],
    "cameras": [
        {
            "position": [0, 0, 3000],
            "direction": [0, 0, -1],
            "up": [0, 1, 0],
            "width": 800,
            "height": 600,
            "field_of_view": 90
        }
    ]
}`;
    parseAndRender();
}

// 窗口大小调整时更新渲染器
window.addEventListener('resize', function() {
    camera.aspect = document.getElementById('scene-container').clientWidth /
        document.getElementById('scene-container').clientHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(
        document.getElementById('scene-container').clientWidth,
        document.getElementById('scene-container').clientHeight
    );
});

// 初始化
window.addEventListener('load', function() {
    initScene();
    parseAndRender();

    // 添加事件监听
    confirmBtn.addEventListener('click', parseAndRender);
    resetBtn.addEventListener('click', resetConfig);
});