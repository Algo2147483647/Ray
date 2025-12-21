// Global variables
let sceneData = null;
let scene = null;
let camera = null;
let renderer = null;
let controls = null;
const objects = [];
let selectedObjectIndex = -1; // Added: Track selected object index
let isJsonPanelCollapsed = false; // Added: Track if JSON panel is collapsed

// Shape parameter configuration will be loaded from external file during initialization
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
const toggleJsonBtn = document.getElementById('toggle-json-btn'); // Added: Toggle button
const jsonPanel = document.getElementById('json-panel'); // Added: JSON panel
const scenePanel = document.getElementById('scene-panel'); // Added: Scene panel
const contentContainer = document.querySelector('.content'); // Added: Content container
const jsonPanelToggle = document.getElementById('json-panel-toggle'); // Added: Bookmark button

// Initialize Three.js scene
function initScene() {
    // Create scene
    scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0a0a1a);

    // Create camera
    camera = new THREE.PerspectiveCamera(75,
        document.getElementById('scene-container').clientWidth /
        document.getElementById('scene-container').clientHeight,
        0.1, 10000);
    camera.position.set(0, 0, 3000);

    // Create renderer
    renderer = new THREE.WebGLRenderer({
        canvas: document.getElementById('scene-canvas'),
        antialias: true
    });
    renderer.setSize(
        document.getElementById('scene-container').clientWidth,
        document.getElementById('scene-container').clientHeight
    );

    // Add orbit controls
    controls = new THREE.OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.05;

    // Add ambient light
    const ambientLight = new THREE.AmbientLight(0x404040, 1.5);
    scene.add(ambientLight);

    // Add directional light
    const directionalLight = new THREE.DirectionalLight(0xffffff, 1);
    directionalLight.position.set(1, 1, 1);
    scene.add(directionalLight);

    // Add axis helper
    const axesHelper = new THREE.AxesHelper(1000);
    scene.add(axesHelper);

    // Start animation loop
    animate();
}

// Animation loop
function animate() {
    requestAnimationFrame(animate);
    controls.update();
    renderer.render(scene, camera);
}

// Load shape parameter configuration
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
    
    // Update renderer size
    setTimeout(() => {
        camera.aspect = document.getElementById('scene-container').clientWidth /
            document.getElementById('scene-container').clientHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(
            document.getElementById('scene-container').clientWidth,
            document.getElementById('scene-container').clientHeight
        );
    }, 300); // Wait for transition animation to complete
}

// Parse JSON and render the scene
async function parseAndRender() {
    // Ensure shape parameters are loaded
    if (Object.keys(SHAPE_PARAMETERS).length === 0) {
        await loadShapeParameters();
    }
    
    try {
        // Parse JSON
        sceneData = JSON.parse(jsonInput.value);

        // Update scene
        updateSceneOnly();

        // Generate parameter table
        generateObjectsTable();

    } catch (e) {
        alert('JSON parse error: ' + e.message);
        console.error(e);
    }
}

// Get parameter list for a shape
function getShapeParameters(shape) {
    const params = SHAPE_PARAMETERS[shape];
    if (!params) return [];
    
    // Get parameter names from keys
    return Object.keys(params);
}

// Generate parameter input controls
function generateParameterInput(paramName, paramType, currentValue) {
    switch (paramType) {
        case '1': // Single number
            return `<input type="number" class="obj-param-${paramName}" value="${currentValue || 0}" step="0.1">`;
        case 'n': // N-dimensional vector (assuming 3D vector here)
            const x = currentValue && currentValue[0] !== undefined ? currentValue[0] : 0;
            const y = currentValue && currentValue[1] !== undefined ? currentValue[1] : 0;
            const z = currentValue && currentValue[2] !== undefined ? currentValue[2] : 0;
            return `
                <input type="number" class="obj-${paramName}-x" value="${x}" step="0.1">
                <input type="number" class="obj-${paramName}-y" value="${y}" step="0.1">
                <input type="number" class="obj-${paramName}-z" value="${z}" step="0.1">
            `;
        case 'text': // Text box
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
        // Get icon based on shape type
        let icon = '◆'; // Default icon
        if (obj.shape === 'cuboid') icon = '◼';
        else if (obj.shape === 'sphere') icon = '●';
        else if (obj.shape === 'triangle') icon = '△';
        else if (obj.shape === 'plane') icon = '▭';
        else if (obj.shape === 'quadratic equation') icon = 'QE';
        else if (obj.shape === 'four-order equation') icon = 'FE';

        // Add class for selected state
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
        
        // Display geometry parameters (fully based on JSON configuration)
        const shapeParams = SHAPE_PARAMETERS[obj.shape];
        if (shapeParams) {
            Object.keys(shapeParams).forEach(param => {
                const paramType = shapeParams[param];
                tableHTML += `<div>`;
                tableHTML += `<span class="geometry-parameter-name">${param}:</span>`;
                tableHTML += `<span class="geometry-parameter-inputs">`;
                
                // Generic handling
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

    // Add event listeners
    document.querySelectorAll('.object-row').forEach(row => {
        const index = parseInt(row.getAttribute('data-index'));
        
        // Add row click event listener
        row.addEventListener('click', function(e) {
            // Prevent event from bubbling to child elements
            if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'SELECT' && e.target.tagName !== 'BUTTON') {
                selectObject(index);
            }
        });
        
        // Add event listener for shape selector
        const shapeSelect = row.querySelector('.obj-shape-select');
        shapeSelect.addEventListener('change', function() {
            // Save current form data first
            saveCurrentFormData();
            
            const newShape = this.value;
            const obj = sceneData.objects[index];
            obj.shape = newShape;
            
            // Update JSON input box
            jsonInput.value = JSON.stringify(sceneData, null, 2);
            
            // Regenerate table
            generateObjectsTable();
            
            // Re-render scene
            parseAndRender();
        });
        
        // Create hover action box
        const actionBox = document.createElement('div');
        actionBox.className = 'action-box';
        actionBox.innerHTML = `
            <button class="move-up-btn" data-index="${index}" ${index === 0 ? 'disabled' : ''}>↑</button>
            <button class="move-down-btn" data-index="${index}" ${index === sceneData.objects.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="delete-btn" data-index="${index}">✕</button>
        `;
        row.appendChild(actionBox);
        
        // Add event listeners for each action button
        actionBox.querySelector('.move-up-btn').addEventListener('click', function(e) {
            e.stopPropagation();
            // Save current form data first
            saveCurrentFormData();
            moveObject(index, -1);
        });
        
        actionBox.querySelector('.move-down-btn').addEventListener('click', function(e) {
            e.stopPropagation();
            // Save current form data first
            saveCurrentFormData();
            moveObject(index, 1);
        });
        
        actionBox.querySelector('.delete-btn').addEventListener('click', function(e) {
            e.stopPropagation();
            // Save current form data first
            saveCurrentFormData();
            deleteObject(index);
        });
    });

    document.getElementById('update-all-btn').addEventListener('click', updateAllObjects);
}

// Added: Function to save current form data
function saveCurrentFormData() {
    if (!sceneData || !sceneData.objects) return;
    
    const rows = objectsTable.querySelectorAll('tbody tr');
    rows.forEach(row => {
        const index = parseInt(row.getAttribute('data-index'));
        const obj = sceneData.objects[index];
        
        if (!obj) return;
        
        // Update ID
        const idInput = row.querySelector('.obj-id');
        if (idInput) obj.id = idInput.value;
        
        // Update shape
        const shapeSelect = row.querySelector('.obj-shape-select');
        if (shapeSelect) obj.shape = shapeSelect.value;
        
        // Generic parameter update method (fully based on JSON configuration)
        const shapeParams = SHAPE_PARAMETERS[obj.shape];
        if (shapeParams) {
            Object.keys(shapeParams).forEach(param => {
                const paramType = shapeParams[param];
                switch (paramType) {
                    case '1':
                        const input = row.querySelector(`.obj-param-${param}`);
                        if (input) {
                            const value = parseFloat(input.value);
                            if (!isNaN(value)) obj[param] = value;
                        }
                        break;
                    case 'n':
                        const xInput = row.querySelector(`.obj-${param}-x`);
                        const yInput = row.querySelector(`.obj-${param}-y`);
                        const zInput = row.querySelector(`.obj-${param}-z`);
                        
                        if (xInput && yInput && zInput) {
                            const x = parseFloat(xInput.value) || 0;
                            const y = parseFloat(yInput.value) || 0;
                            const z = parseFloat(zInput.value) || 0;
                            obj[param] = [x, y, z];
                        }
                        break;
                    case 'text':
                        const textInput = row.querySelector(`.obj-param-${param}`);
                        if (textInput) {
                            try {
                                obj[param] = JSON.parse(textInput.value);
                            } catch (e) {
                                // If not valid JSON, save as string
                                obj[param] = textInput.value;
                            }
                        }
                        break;
                }
            });
        }
        
        // Update material ID
        const materialIdInput = row.querySelector('.obj-material-id');
        if (materialIdInput) obj.material_id = materialIdInput.value;
    });
}

// Added: Function to update scene without regenerating table
function updateSceneOnly() {
    if (!sceneData || !sceneData.objects) return;

    // Remove previous objects
    objects.forEach(obj => scene.remove(obj));
    objects.length = 0;

    // Create geometries
    let cuboids = 0;
    let spheres = 0;
    let lights = 0;

    sceneData.objects.forEach(obj => {
        let geometry, material, mesh;

        // Set color based on material or selection state
        let color = 0xffffff; // Default white
        
        // If there is a selected object and current object is the selected one, set to red
        if (selectedObjectIndex !== -1 && sceneData.objects[selectedObjectIndex] === obj) {
            color = 0xff0000; // Red
        } else {
            // Otherwise set color based on material
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
}

// Added: Function to select object
function selectObject(index) {
    // Save current form data
    saveCurrentFormData();
    
    // Update selected index
    selectedObjectIndex = index;
    
    // Update JSON input box to keep in sync with sceneData
    jsonInput.value = JSON.stringify(sceneData, null, 2);
    
    // Update scene only without regenerating table
    updateSceneOnly();
    
    // Update selected state of table rows
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
    
    // Check boundaries
    if (newIndex < 0 || newIndex >= sceneData.objects.length) return;
    
    // Swap object positions
    const temp = sceneData.objects[index];
    sceneData.objects[index] = sceneData.objects[newIndex];
    sceneData.objects[newIndex] = temp;
    
    // Update JSON input box
    jsonInput.value = JSON.stringify(sceneData, null, 2);
    
    // Regenerate table
    generateObjectsTable();
    
    // Update scene only
    updateSceneOnly();
}

// Delete object
function deleteObject(index) {
    if (!sceneData || !sceneData.objects) return;
    
    // Confirm deletion
    const obj = sceneData.objects[index];
    if (!confirm(`Are you sure you want to delete object "${obj.id}"?`)) {
        return;
    }
    
    // Remove object from array
    sceneData.objects.splice(index, 1);
    
    // Update JSON input box
    jsonInput.value = JSON.stringify(sceneData, null, 2);
    
    // Regenerate table
    generateObjectsTable();
    
    // Update scene only
    updateSceneOnly();
}

// Update all objects
function updateAllObjects() {
    if (!sceneData || !sceneData.objects) return;
    
    // Directly get latest data from form
    saveCurrentFormData();
    
    // Update JSON input box
    jsonInput.value = JSON.stringify(sceneData, null, 2);
    
    // Update scene only
    updateSceneOnly();
}

// Reset configuration
function resetConfig() {
    // Reset selected index
    selectedObjectIndex = -1;
    
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

// Update renderer when window size changes
window.addEventListener('resize', function() {
    camera.aspect = document.getElementById('scene-container').clientWidth /
        document.getElementById('scene-container').clientHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(
        document.getElementById('scene-container').clientWidth,
        document.getElementById('scene-container').clientHeight
    );
});

// Initialize
window.addEventListener('load', async function() {
    await loadShapeParameters(); // First load shape parameters
    initScene();
    parseAndRender();

    // Add event listeners
    confirmBtn.addEventListener('click', parseAndRender);
    resetBtn.addEventListener('click', resetConfig);
    toggleJsonBtn.addEventListener('click', toggleJsonPanel); // Added: Toggle button event listener
    jsonPanelToggle.addEventListener('click', toggleJsonPanel); // Added: Bookmark button event listener
});