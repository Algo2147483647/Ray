// Global variables
let sceneData = null;
let scene = null;
let camera = null;
let renderer = null;
let controls = null;
const objects = [];

// 定义形状及其参数的映射关系
const SHAPE_PARAMETERS = {
    'cuboid': {
        icon: '◼',
        params: ['position', 'size']
    },
    'sphere': {
        icon: '●',
        params: ['position', 'r']
    },
    'triangle': {
        icon: '△',
        params: ['p1', 'p2', 'p3']
    },
    'plane': {
        icon: '▭',
        params: ['A', 'b']
    },
    'quadratic equation': {
        icon: 'QE',
        params: ['a', 'b', 'c']
    },
    'four-order equation': {
        icon: 'FE',
        params: ['a']
    }
};

// DOM elements
const jsonInput = document.getElementById('json-input');
const confirmBtn = document.getElementById('confirm-btn');
const resetBtn = document.getElementById('reset-btn');
const objectsTable = document.getElementById('objects-table');
const cuboidCount = document.getElementById('cuboid-count');
const sphereCount = document.getElementById('sphere-count');
const totalCount = document.getElementById('total-count');
const lightCount = document.getElementById('light-count');

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

// Parse JSON and render the scene
function parseAndRender() {
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

            // Set color based on material
            let color = 0xffffff;
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

            // Create material
            material = new THREE.MeshPhongMaterial({
                color: color,
                transparent: obj.material_id === 'Glass',
                opacity: obj.material_id === 'Glass' ? 0.7 : 1.0,
                wireframe: obj.id === 'WorldBox'
            });

            // Create geometry
            if (obj.shape === 'cuboid') {
                geometry = new THREE.BoxGeometry(obj.size[0], obj.size[1], obj.size[2]);
                cuboids++;
            } else if (obj.shape === 'sphere') {
                const radius = obj.r || obj.radius || 100; // 支持r和radius两种字段
                geometry = new THREE.SphereGeometry(radius, 32, 32);
                spheres++;
            } else if (obj.shape === 'triangle') {
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
                // Triangles are not counted in stats
            }

            // Create mesh
            if (geometry) {
                mesh = new THREE.Mesh(geometry, material);
                mesh.position.set(obj.position ? obj.position[0] : 0, 
                                  obj.position ? obj.position[1] : 0, 
                                  obj.position ? obj.position[2] : 0);
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
        tableHTML += `
            <tr data-index="${index}" class="object-row">
                <td><input type="text" class="obj-id" value="${obj.id || ''}"></td>
                <td>
                    <select class="obj-shape-select">
                        ${Object.keys(SHAPE_PARAMETERS).map(shape => 
                            `<option value="${shape}" ${obj.shape === shape ? 'selected' : ''}>${shape}</option>`
                        ).join('')}
                    </select>
                    <span class="shape-icon">${SHAPE_PARAMETERS[obj.shape]?.icon || '◆'}</span>
                </td>
                <td>
        `;
        
        // 显示几何参数
        const shapeParams = SHAPE_PARAMETERS[obj.shape] ? SHAPE_PARAMETERS[obj.shape].params : [];
        shapeParams.forEach(param => {
            switch(param) {
                case 'position':
                    tableHTML += `
                        Position: 
                        <input type="number" class="obj-pos-x" value="${obj.position ? obj.position[0] : 0}" step="10">
                        <input type="number" class="obj-pos-y" value="${obj.position ? obj.position[1] : 0}" step="10">
                        <input type="number" class="obj-pos-z" value="${obj.position ? obj.position[2] : 0}" step="10">
                        <br/>
                    `;
                    break;
                    
                case 'size':
                    tableHTML += `
                        Size: 
                        <input type="number" class="obj-size-w" value="${obj.size ? obj.size[0] : 100}" min="1" step="10">
                        <input type="number" class="obj-size-h" value="${obj.size ? obj.size[1] : 100}" min="1" step="10">
                        <input type="number" class="obj-size-d" value="${obj.size ? obj.size[2] : 100}" min="1" step="10">
                        <br/>
                    `;
                    break;
                    
                case 'r':
                    const radius = obj.r || obj.radius || 100;
                    tableHTML += `
                        Radius: 
                        <input type="number" class="obj-radius" value="${radius}" min="1" step="10">
                        <br/>
                    `;
                    break;
                    
                case 'p1':
                    tableHTML += `
                        P1: 
                        <input type="number" class="obj-p1-x" value="${obj.p1 ? obj.p1[0] : 0}" step="0.1">
                        <input type="number" class="obj-p1-y" value="${obj.p1 ? obj.p1[1] : 0}" step="0.1">
                        <input type="number" class="obj-p1-z" value="${obj.p1 ? obj.p1[2] : 0}" step="0.1">
                        <br/>
                    `;
                    break;
                    
                case 'p2':
                    tableHTML += `
                        P2: 
                        <input type="number" class="obj-p2-x" value="${obj.p2 ? obj.p2[0] : 0}" step="0.1">
                        <input type="number" class="obj-p2-y" value="${obj.p2 ? obj.p2[1] : 0}" step="0.1">
                        <input type="number" class="obj-p2-z" value="${obj.p2 ? obj.p2[2] : 0}" step="0.1">
                        <br/>
                    `;
                    break;
                    
                case 'p3':
                    tableHTML += `
                        P3: 
                        <input type="number" class="obj-p3-x" value="${obj.p3 ? obj.p3[0] : 0}" step="0.1">
                        <input type="number" class="obj-p3-y" value="${obj.p3 ? obj.p3[1] : 0}" step="0.1">
                        <input type="number" class="obj-p3-z" value="${obj.p3 ? obj.p3[2] : 0}" step="0.1">
                        <br/>
                    `;
                    break;
                    
                case 'A':
                    tableHTML += `
                        A: 
                        <input type="number" class="obj-A-x" value="${obj.A ? obj.A[0] : 0}" step="0.1">
                        <input type="number" class="obj-A-y" value="${obj.A ? obj.A[1] : 0}" step="0.1">
                        <input type="number" class="obj-A-z" value="${obj.A ? obj.A[2] : 0}" step="0.1">
                        <br/>
                    `;
                    break;
                    
                case 'b':
                    tableHTML += `
                        b: 
                        <input type="number" class="obj-b" value="${obj.b || 0}" step="0.1">
                        <br/>
                    `;
                    break;
                    
                case 'a':
                    if (obj.shape === 'quadratic equation') {
                        tableHTML += `
                            a (3x3 matrix): 
                            <textarea class="obj-a-quadratic" rows="3" cols="30">${obj.a ? JSON.stringify(obj.a) : '[[0,0,0],[0,0,0],[0,0,0]]'}</textarea>
                            <br/>
                        `;
                    } else if (obj.shape === 'four-order equation') {
                        tableHTML += `
                            a (coefficients): 
                            <textarea class="obj-a-fourth" rows="3" cols="30">${obj.a ? JSON.stringify(obj.a) : '[]'}</textarea>
                            <br/>
                        `;
                    }
                    break;
                    
                case 'c':
                    tableHTML += `
                        c: 
                        <input type="number" class="obj-c" value="${obj.c || 0}" step="0.1">
                        <br/>
                    `;
                    break;
            }
        });

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
        
        // 更新位置和尺寸
        if (obj.shape === 'cuboid') {
            const posX = parseFloat(row.querySelector('.obj-pos-x').value) || 0;
            const posY = parseFloat(row.querySelector('.obj-pos-y').value) || 0;
            const posZ = parseFloat(row.querySelector('.obj-pos-z').value) || 0;
            obj.position = [posX, posY, posZ];
            
            const sizeW = parseFloat(row.querySelector('.obj-size-w').value) || 1;
            const sizeH = parseFloat(row.querySelector('.obj-size-h').value) || 1;
            const sizeD = parseFloat(row.querySelector('.obj-size-d').value) || 1;
            obj.size = [sizeW, sizeH, sizeD];
        } else if (obj.shape === 'sphere') {
            const posX = parseFloat(row.querySelector('.obj-pos-x').value) || 0;
            const posY = parseFloat(row.querySelector('.obj-pos-y').value) || 0;
            const posZ = parseFloat(row.querySelector('.obj-pos-z').value) || 0;
            obj.position = [posX, posY, posZ];
            
            const radius = parseFloat(row.querySelector('.obj-radius').value) || 1;
            obj.r = radius;
            if (obj.radius !== undefined) {
                obj.radius = radius;
            }
        } else if (obj.shape === 'triangle') {
            const p1x = parseFloat(row.querySelector('.obj-p1-x').value) || 0;
            const p1y = parseFloat(row.querySelector('.obj-p1-y').value) || 0;
            const p1z = parseFloat(row.querySelector('.obj-p1-z').value) || 0;
            obj.p1 = [p1x, p1y, p1z];
            
            const p2x = parseFloat(row.querySelector('.obj-p2-x').value) || 0;
            const p2y = parseFloat(row.querySelector('.obj-p2-y').value) || 0;
            const p2z = parseFloat(row.querySelector('.obj-p2-z').value) || 0;
            obj.p2 = [p2x, p2y, p2z];
            
            const p3x = parseFloat(row.querySelector('.obj-p3-x').value) || 0;
            const p3y = parseFloat(row.querySelector('.obj-p3-y').value) || 0;
            const p3z = parseFloat(row.querySelector('.obj-p3-z').value) || 0;
            obj.p3 = [p3x, p3y, p3z];
        } else if (obj.shape === 'plane') {
            const ax = parseFloat(row.querySelector('.obj-A-x').value) || 0;
            const ay = parseFloat(row.querySelector('.obj-A-y').value) || 0;
            const az = parseFloat(row.querySelector('.obj-A-z').value) || 0;
            obj.A = [ax, ay, az];
            
            const b = parseFloat(row.querySelector('.obj-b').value) || 0;
            obj.b = b;
        } else if (obj.shape === 'quadratic equation') {
            const posX = parseFloat(row.querySelector('.obj-pos-x').value) || 0;
            const posY = parseFloat(row.querySelector('.obj-pos-y').value) || 0;
            const posZ = parseFloat(row.querySelector('.obj-pos-z').value) || 0;
            obj.position = [posX, posY, posZ];
            
            try {
                const aText = row.querySelector('.obj-a-quadratic').value;
                obj.a = JSON.parse(aText);
            } catch (e) {
                console.warn("Failed to parse quadratic equation coefficients");
            }
            
            const c = parseFloat(row.querySelector('.obj-c').value) || 0;
            obj.c = c;
        } else if (obj.shape === 'four-order equation') {
            try {
                const aText = row.querySelector('.obj-a-fourth').value;
                obj.a = JSON.parse(aText);
            } catch (e) {
                console.warn("Failed to parse fourth-order equation coefficients");
            }
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
window.addEventListener('load', function() {
    initScene();
    parseAndRender();

    // 添加事件监听
    confirmBtn.addEventListener('click', parseAndRender);
    resetBtn.addEventListener('click', resetConfig);
});