// Global variables
let sceneData = null;
let scene = null;
let camera = null;
let renderer = null;
let controls = null;
const objects = [];
let selectedObjectIndex = -1; // 新增：跟踪选中的对象索引
let isJsonPanelCollapsed = false; // 新增：跟踪JSON面板是否折叠

// 形状参数配置将在初始化时从外部文件加载
let SHAPE_PARAMETERS = {};

// DOM elements
const jsonInput = document.getElementById('json-input');
const confirmBtn = document.getElementById('confirm-btn');
const resetBtn = document.getElementById('reset-btn');
const objectsTable = document.getElementById('objects-table');
const cuboidCount = document.getElementById('cuboid-count');
const sphereCount = document.getElementById('sphere-count');
const totalCount = document.getElementById('total-count');
const lightCount = document.getElementById('light-count');
const toggleJsonBtn = document.getElementById('toggle-json-btn'); // 新增：切换按钮
const jsonPanel = document.getElementById('json-panel'); // 新增：JSON面板
const scenePanel = document.getElementById('scene-panel'); // 新增：场景面板
const contentContainer = document.querySelector('.content'); // 新增：内容容器
const jsonPanelToggle = document.getElementById('json-panel-toggle'); // 新增：书签按钮

// Initialize Three.js scene
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

// Animation loop
function animate() {
    requestAnimationFrame(animate);
    controls.update();
    renderer.render(scene, camera);
}

// 加载形状参数配置
async function loadShapeParameters() {
    try {
        const response = await fetch('shape_parameters.json');
        SHAPE_PARAMETERS = await response.json();
        console.log('Shape parameters loaded:', SHAPE_PARAMETERS);
    } catch (error) {
        console.error('Failed to load shape parameters:', error);
    }
}

// Toggle JSON panel visibility
function toggleJsonPanel() {
    isJsonPanelCollapsed = !isJsonPanelCollapsed;
    
    if (isJsonPanelCollapsed) {
        jsonPanel.classList.add('collapsed');
        contentContainer.classList.add('full-width-scene');
        jsonPanelToggle.textContent = '❯';
    } else {
        jsonPanel.classList.remove('collapsed');
        contentContainer.classList.remove('full-width-scene');
        jsonPanelToggle.textContent = '❮';
    }
    
    // 更新渲染器尺寸
    setTimeout(() => {
        camera.aspect = document.getElementById('scene-container').clientWidth /
            document.getElementById('scene-container').clientHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(
            document.getElementById('scene-container').clientWidth,
            document.getElementById('scene-container').clientHeight
        );
    }, 300); // 等待过渡动画完成
}

// Parse JSON and render the scene
async function parseAndRender() {
    // 确保形状参数已加载
    if (Object.keys(SHAPE_PARAMETERS).length === 0) {
        await loadShapeParameters();
    }
    
    try {
        // 解析JSON
        sceneData = JSON.parse(jsonInput.value);

        // Remove previous objects
        objects.forEach(obj => scene.remove(obj));
        objects.length = 0;

        // 创建几何体
        let cuboids = 0;
        let spheres = 0;
        let lights = 0;

        sceneData.objects.forEach(obj => {
            let geometry, material, mesh;

            // Set color based on material or selection state
            let color = 0xffffff; // 默认白色
            
            // 如果有选中的对象且当前对象是选中的对象，则设为红色
            if (selectedObjectIndex !== -1 && sceneData.objects[selectedObjectIndex] === obj) {
                color = 0xff0000; // 红色
            } else {
                // 否则根据材质设置颜色
                const materialInfo = sceneData.materials.find(m => m.id === obj.material_id);
                if (materialInfo) {
                    // Convert color values from 0-1 range to 0-255 and combine to hex
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
            }

            // Create material
            material = new THREE.MeshPhongMaterial({
                color: color,
                transparent: obj.material_id === 'Glass',
                opacity: obj.material_id === 'Glass' ? 0.7 : 1.0,
                wireframe: obj.id === 'WorldBox'
            });

            // Create geometry based on shape parameters configuration
            const shapeParams = SHAPE_PARAMETERS[obj.shape];
            if (shapeParams) {
                if (obj.shape === 'cuboid' && obj.size) {
                    geometry = new THREE.BoxGeometry(obj.size[0], obj.size[1], obj.size[2]);
                    cuboids++;
                } else if (obj.shape === 'sphere') {
                    const radius = obj.r || obj.radius || 100;
                    geometry = new THREE.SphereGeometry(radius, 32, 32);
                    spheres++;
                } else if (obj.shape === 'triangle' && obj.p1 && obj.p2 && obj.p3) {
                    // Create triangle geometry
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
                }
            }

            // Create mesh
            if (geometry) {
                mesh = new THREE.Mesh(geometry, material);
                // Handle position for all shapes
                if (obj.position) {
                    mesh.position.set(obj.position[0], obj.position[1], obj.position[2]);
                } else {
                    mesh.position.set(0, 0, 0);
                }
                scene.add(mesh);
                objects.push(mesh);
            }
        });

        // Update statistics (only count cuboids and spheres)
        cuboidCount.textContent = cuboids;
        sphereCount.textContent = spheres;
        totalCount.textContent = cuboids + spheres;
        lightCount.textContent = lights;

        // 生成参数表格
        generateObjectsTable();

    } catch (e) {
        alert('JSON parse error: ' + e.message);
        console.error(e);
    }
}

// 获取形状的参数列表
function getShapeParameters(shape) {
    const params = SHAPE_PARAMETERS[shape];
    if (!params) return [];
    
    // 从键中获取参数名
    return Object.keys(params);
}

// 生成参数输入控件
function generateParameterInput(paramName, paramType, currentValue) {
    switch (paramType) {
        case '1': // 单个数字
            return `<input type="number" class="obj-param-${paramName}" value="${currentValue || 0}" step="0.1">`;
        case 'n': // N维向量 (这里假设是3D向量)
            const x = currentValue && currentValue[0] !== undefined ? currentValue[0] : 0;
            const y = currentValue && currentValue[1] !== undefined ? currentValue[1] : 0;
            const z = currentValue && currentValue[2] !== undefined ? currentValue[2] : 0;
            return `
                <input type="number" class="obj-${paramName}-x" value="${x}" step="0.1">
                <input type="number" class="obj-${paramName}-y" value="${y}" step="0.1">
                <input type="number" class="obj-${paramName}-z" value="${z}" step="0.1">
            `;
        case 'text': // 文本框
            return `<textarea class="obj-param-${paramName}" rows="3" cols="30">${currentValue ? JSON.stringify(currentValue) : ''}</textarea>`;
        default:
            return `<input type="text" class="obj-param-${paramName}" value="${currentValue || ''}">`;
    }
}

// Generate geometry parameters table
function generateObjectsTable() {
    if (!sceneData || !sceneData.objects) return;
    
    let tableHTML = `
        <table>
            <thead>
                <tr>
                <th>ID</th>
                <th>Shape</th>
                <th>Geometry Parameters</th>
                <th>Material ID</th>
            </tr>
        </thead>
        <tbody>
        `;

    sceneData.objects.forEach((obj, index) => {
        // 根据形状类型获取图标
        let icon = '◆'; // 默认图标
        if (obj.shape === 'cuboid') icon = '◼';
        else if (obj.shape === 'sphere') icon = '●';
        else if (obj.shape === 'triangle') icon = '△';
        else if (obj.shape === 'plane') icon = '▭';
        else if (obj.shape === 'quadratic equation') icon = 'QE';
        else if (obj.shape === 'four-order equation') icon = 'FE';

        // 添加选中状态的类
        const isSelected = index === selectedObjectIndex;
        const rowClass = isSelected ? 'object-row selected' : 'object-row';

        tableHTML += `
            <tr data-index="${index}" class="${rowClass}">
                <td><input type="text" class="obj-id" value="${obj.id || ''}"></td>
                <td>
                    <select class="obj-shape-select">
                        ${Object.keys(SHAPE_PARAMETERS).map(shape => 
                            `<option value="${shape}" ${obj.shape === shape ? 'selected' : ''}>${shape}</option>`
                        ).join('')}
                    </select>
                    <span class="shape-icon">${icon}</span>
                </td>
                <td class="geometry-parameters">
        `;
        
        // 显示几何参数 (完全基于JSON配置)
        const shapeParams = SHAPE_PARAMETERS[obj.shape];
        if (shapeParams) {
            Object.keys(shapeParams).forEach(param => {
                const paramType = shapeParams[param];
                tableHTML += `<div>`;
                tableHTML += `<span class="geometry-parameter-name">${param}:</span>`;
                tableHTML += `<span class="geometry-parameter-inputs">`;
                
                // 通用处理方式
                tableHTML += generateParameterInput(param, paramType, obj[param]);
                tableHTML += `</span></div>`;
            });
        }

        tableHTML += `
                </td>
                <td>
                    <input type="text" class="obj-material-id" value="${obj.material_id || obj.material || ''}">
                </td>
            </tr>
            `;
    });

    tableHTML += `
            </tbody>
        </table>
        <div class="table-controls">
            <button id="update-all-btn" class="btn">Update All Objects</button>
        </div>
        `;

    objectsTable.innerHTML = tableHTML;

    // 添加事件监听器
    document.querySelectorAll('.object-row').forEach(row => {
        const index = parseInt(row.getAttribute('data-index'));
        
        // 添加行点击事件监听器
        row.addEventListener('click', function(e) {
            // 阻止事件冒泡到子元素
            if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'SELECT' && e.target.tagName !== 'BUTTON') {
                selectObject(index);
            }
        });
        
        // 为形状选择器添加事件监听
        const shapeSelect = row.querySelector('.obj-shape-select');
        shapeSelect.addEventListener('change', function() {
            const newShape = this.value;
            const obj = sceneData.objects[index];
            obj.shape = newShape;
            
            // 更新JSON输入框
            jsonInput.value = JSON.stringify(sceneData, null, 2);
            
            // 重新生成表格
            generateObjectsTable();
            
            // 重新渲染场景
            parseAndRender();
        });
        
        // 创建悬停操作框
        const actionBox = document.createElement('div');
        actionBox.className = 'action-box';
        actionBox.innerHTML = `
            <button class="move-up-btn" data-index="${index}" ${index === 0 ? 'disabled' : ''}>↑</button>
            <button class="move-down-btn" data-index="${index}" ${index === sceneData.objects.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="delete-btn" data-index="${index}">✕</button>
        `;
        row.appendChild(actionBox);
        
        // 为每个操作按钮添加事件监听器
        actionBox.querySelector('.move-up-btn').addEventListener('click', function(e) {
            e.stopPropagation();
            moveObject(index, -1);
        });
        
        actionBox.querySelector('.move-down-btn').addEventListener('click', function(e) {
            e.stopPropagation();
            moveObject(index, 1);
        });
        
        actionBox.querySelector('.delete-btn').addEventListener('click', function(e) {
            e.stopPropagation();
            deleteObject(index);
        });
    });

    document.getElementById('update-all-btn').addEventListener('click', updateAllObjects);
}

// 新增：选中对象函数
function selectObject(index) {
    // 更新选中索引
    selectedObjectIndex = index;
    
    // 重新渲染场景以应用颜色变化
    parseAndRender();
    
    // 更新表格行的选中状态
    document.querySelectorAll('.object-row').forEach(row => {
        const rowIndex = parseInt(row.getAttribute('data-index'));
        if (rowIndex === index) {
            row.classList.add('selected');
        } else {
            row.classList.remove('selected');
        }
    });
}

// Move object
function moveObject(index, direction) {
    if (!sceneData || !sceneData.objects) return;
    
    const newIndex = index + direction;
    
    // 检查边界
    if (newIndex < 0 || newIndex >= sceneData.objects.length) return;
    
    // 交换对象位置
    const temp = sceneData.objects[index];
    sceneData.objects[index] = sceneData.objects[newIndex];
    sceneData.objects[newIndex] = temp;
    
    // 更新JSON输入框
    jsonInput.value = JSON.stringify(sceneData, null, 2);
    
    // 重新生成表格
    generateObjectsTable();
    
    // 重新渲染场景
    parseAndRender();
}

// Delete object
function deleteObject(index) {
    if (!sceneData || !sceneData.objects) return;
    
    // Confirm deletion
    const obj = sceneData.objects[index];
    if (!confirm(`Are you sure you want to delete object "${obj.id}"?`)) {
        return;
    }
    
    // 从数组中移除对象
    sceneData.objects.splice(index, 1);
    
    // 更新JSON输入框
    jsonInput.value = JSON.stringify(sceneData, null, 2);
    
    // 重新生成表格
    generateObjectsTable();
    
    // 重新渲染场景
    parseAndRender();
}

// Update all objects
function updateAllObjects() {
    if (!sceneData || !sceneData.objects) return;
    
    const rows = objectsTable.querySelectorAll('tbody tr');
    
    rows.forEach(row => {
        const index = parseInt(row.getAttribute('data-index'));
        const obj = sceneData.objects[index];
        
        // 更新ID
        const idInput = row.querySelector('.obj-id');
        if (idInput) obj.id = idInput.value;
        
        // 更新形状
        const shapeSelect = row.querySelector('.obj-shape-select');
        if (shapeSelect) obj.shape = shapeSelect.value;
        
        // 通用参数更新方法 (完全基于JSON配置)
        const shapeParams = SHAPE_PARAMETERS[obj.shape];
        if (shapeParams) {
            Object.keys(shapeParams).forEach(param => {
                const paramType = shapeParams[param];
                switch (paramType) {
                    case '1':
                        const value = parseFloat(row.querySelector(`.obj-param-${param}`).value) || 0;
                        obj[param] = value;
                        break;
                    case 'n':
                        const x = parseFloat(row.querySelector(`.obj-${param}-x`).value) || 0;
                        const y = parseFloat(row.querySelector(`.obj-${param}-y`).value) || 0;
                        const z = parseFloat(row.querySelector(`.obj-${param}-z`).value) || 0;
                        obj[param] = [x, y, z];
                        break;
                    case 'text':
                        try {
                            const text = row.querySelector(`.obj-param-${param}`).value;
                            obj[param] = JSON.parse(text);
                        } catch (e) {
                            console.warn(`Failed to parse parameter ${param} as JSON`);
                        }
                        break;
                }
            });
        }
        
        // 更新材质ID
        const materialIdInput = row.querySelector('.obj-material-id');
        if (materialIdInput) obj.material_id = materialIdInput.value;
    });
    
    // 更新JSON输入框
    jsonInput.value = JSON.stringify(sceneData, null, 2);
    
    // 重新渲染场景
    parseAndRender();
}

// Reset configuration
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
window.addEventListener('load', async function() {
    await loadShapeParameters(); // 首先加载形状参数
    initScene();
    parseAndRender();

    // 添加事件监听
    confirmBtn.addEventListener('click', parseAndRender);
    resetBtn.addEventListener('click', resetConfig);
    toggleJsonBtn.addEventListener('click', toggleJsonPanel); // 新增：切换按钮事件监听
    jsonPanelToggle.addEventListener('click', toggleJsonPanel); // 新增：书签按钮事件监听
});