import { useEffect, useMemo, useRef } from "react";
import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import type { SceneDocument, SceneMaterial, SceneObject } from "../types/scene";
import {
  estimateObjectCenter,
  estimateProxyRadius,
  getMaterialForObject,
  getPreviewSupport,
  materialEmitsLight,
  toNumber,
  toVec3
} from "../lib/scene-utils";

interface SceneViewportProps {
  scene: SceneDocument;
  selectedObjectId: string | null;
}

interface ViewportRuntime {
  scene: THREE.Scene;
  camera: THREE.PerspectiveCamera;
  renderer: THREE.WebGLRenderer;
  controls: OrbitControls;
  previewGroup: THREE.Group;
  resizeObserver: ResizeObserver | null;
  frameId: number | null;
}

export function SceneViewport({ scene, selectedObjectId }: SceneViewportProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const runtimeRef = useRef<ViewportRuntime | null>(null);

  const previewWarnings = useMemo(
    () =>
      scene.objects
        .filter((object) => getPreviewSupport(object.shape) === "proxy")
        .map((object) => `${object.id} uses a proxy preview for ${object.shape}.`),
    [scene.objects]
  );

  useEffect(() => {
    const host = hostRef.current;
    if (!host) {
      return;
    }

    const threeScene = new THREE.Scene();
    threeScene.background = new THREE.Color("#0b1220");
    threeScene.fog = new THREE.Fog("#0b1220", 3500, 12000);

    const camera = new THREE.PerspectiveCamera(55, 1, 0.1, 20000);
    camera.position.set(0, 0, 3000);

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    renderer.setSize(host.clientWidth, host.clientHeight);
    host.appendChild(renderer.domElement);

    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;
    controls.minDistance = 100;
    controls.maxDistance = 14000;

    const previewGroup = new THREE.Group();
    threeScene.add(previewGroup);

    const ambient = new THREE.HemisphereLight("#bed0ff", "#0f172a", 1.25);
    const sun = new THREE.DirectionalLight("#fff2ce", 1.35);
    sun.position.set(1800, 2600, 1400);
    threeScene.add(ambient, sun);

    const grid = new THREE.GridHelper(5000, 20, "#465f7d", "#233146");
    grid.position.y = -1000;
    threeScene.add(grid);

    const axes = new THREE.AxesHelper(800);
    threeScene.add(axes);

    const resize = () => {
      const width = host.clientWidth;
      const height = host.clientHeight;
      camera.aspect = width / Math.max(height, 1);
      camera.updateProjectionMatrix();
      renderer.setSize(width, height);
    };

    const resizeObserver =
      typeof ResizeObserver !== "undefined"
        ? new ResizeObserver(resize)
        : null;
    resizeObserver?.observe(host);
    resize();

    const runtime: ViewportRuntime = {
      scene: threeScene,
      camera,
      renderer,
      controls,
      previewGroup,
      resizeObserver,
      frameId: null
    };
    runtimeRef.current = runtime;

    const renderLoop = () => {
      controls.update();
      renderer.render(threeScene, camera);
      runtime.frameId = window.requestAnimationFrame(renderLoop);
    };

    renderLoop();

    return () => {
      if (runtime.frameId) {
        window.cancelAnimationFrame(runtime.frameId);
      }
      resizeObserver?.disconnect();
      previewGroup.clear();
      renderer.dispose();
      host.removeChild(renderer.domElement);
      runtimeRef.current = null;
    };
  }, []);

  useEffect(() => {
    const runtime = runtimeRef.current;
    if (!runtime) {
      return;
    }

    clearGroup(runtime.previewGroup);

    const cameraConfig = scene.cameras[0];
    runtime.camera.fov = toNumber(cameraConfig?.field_of_view, 90);
    runtime.camera.position.fromArray(toVec3(cameraConfig?.position));
    runtime.camera.up.fromArray(toVec3(cameraConfig?.up));
    const cameraTarget = new THREE.Vector3(...toVec3(cameraConfig?.position)).add(
      new THREE.Vector3(...toVec3(cameraConfig?.direction))
    );
    runtime.camera.lookAt(cameraTarget);
    runtime.camera.updateProjectionMatrix();

    let selectedTarget = new THREE.Vector3(0, 0, 0);

    scene.objects.forEach((object) => {
      const material = getMaterialForObject(scene, object);
      const isSelected = object.id === selectedObjectId;
      const renderable = buildRenderableObject(object, material, isSelected);
      runtime.previewGroup.add(renderable);

      if (isSelected) {
        selectedTarget = new THREE.Vector3(...estimateObjectCenter(object));
      }

      if (materialEmitsLight(material) && object.position) {
        const pointLight = new THREE.PointLight(
          toThreeColor(material?.color ?? [1, 1, 1]),
          2.5,
          4500
        );
        pointLight.position.fromArray(toVec3(object.position));
        runtime.previewGroup.add(pointLight);
      }
    });

    runtime.controls.target.copy(selectedTarget);
    runtime.controls.update();
  }, [scene, selectedObjectId]);

  return (
    <div className="panel-card viewport-card">
      <div className="panel-heading viewport-heading">
        <div>
          <p className="eyebrow">Viewport</p>
          <h2>Preview honors camera and material semantics</h2>
        </div>
        <div className="viewport-legend">
          <span className="support-badge exact">Exact</span>
          <span className="support-badge proxy">Proxy</span>
        </div>
      </div>
      <div ref={hostRef} className="viewport-canvas" />
      <div className="viewport-footer">
        <p>Orbit to inspect the scene. Selecting an object recenters the target.</p>
        {previewWarnings.length > 0 && (
          <ul className="warning-list">
            {previewWarnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function buildRenderableObject(
  object: SceneObject,
  materialInfo: SceneMaterial | undefined,
  isSelected: boolean
) {
  const support = getPreviewSupport(object.shape);
  const material = createSurfaceMaterial(materialInfo, isSelected, support === "proxy");

  switch (object.shape) {
    case "cuboid": {
      const size = toVec3(object.size);
      const geometry = new THREE.BoxGeometry(size[0], size[1], size[2]);
      const mesh = new THREE.Mesh(geometry, material);
      mesh.position.fromArray(toVec3(object.position));
      return decorateSelection(mesh, geometry, isSelected);
    }
    case "sphere": {
      const radius = Math.max(1, toNumber(object.r, 100));
      const geometry = new THREE.SphereGeometry(radius, 36, 24);
      const mesh = new THREE.Mesh(geometry, material);
      mesh.position.fromArray(toVec3(object.position));
      return decorateSelection(mesh, geometry, isSelected);
    }
    case "triangle": {
      const geometry = new THREE.BufferGeometry();
      const vertices = new Float32Array([
        ...toVec3(object.p1),
        ...toVec3(object.p2),
        ...toVec3(object.p3)
      ]);
      geometry.setAttribute("position", new THREE.BufferAttribute(vertices, 3));
      geometry.setIndex([0, 1, 2]);
      geometry.computeVertexNormals();
      const mesh = new THREE.Mesh(geometry, material);
      return decorateSelection(mesh, geometry, isSelected);
    }
    case "plane": {
      const geometry = new THREE.PlaneGeometry(3200, 3200, 12, 12);
      const mesh = new THREE.Mesh(
        geometry,
        new THREE.MeshStandardMaterial({
          color: toThreeColor(materialInfo?.color ?? [0.9, 0.9, 0.95]),
          side: THREE.DoubleSide,
          transparent: true,
          opacity: 0.28,
          metalness: 0.1,
          roughness: 0.75
        })
      );
      const normal = new THREE.Vector3(...toVec3(object.A));
      if (normal.lengthSq() === 0) {
        normal.set(0, 1, 0);
      }
      const point = normal
        .clone()
        .multiplyScalar(
          -toNumber(Array.isArray(object.b) ? object.b[0] : object.b) / normal.lengthSq()
        );
      const quaternion = new THREE.Quaternion().setFromUnitVectors(
        new THREE.Vector3(0, 0, 1),
        normal.clone().normalize()
      );
      mesh.position.copy(point);
      mesh.quaternion.copy(quaternion);
      return decorateSelection(mesh, geometry, isSelected);
    }
    case "quadratic equation": {
      return buildProxyVolume(object, materialInfo, isSelected, "icosahedron");
    }
    case "four-order equation": {
      return buildProxyVolume(object, materialInfo, isSelected, "octahedron");
    }
  }
}

function buildProxyVolume(
  object: SceneObject,
  materialInfo: SceneMaterial | undefined,
  isSelected: boolean,
  kind: "icosahedron" | "octahedron"
) {
  const radius = estimateProxyRadius(object);
  const geometry =
    kind === "icosahedron"
      ? new THREE.IcosahedronGeometry(radius, 1)
      : new THREE.OctahedronGeometry(radius, 1);
  const wireframe = new THREE.LineSegments(
    new THREE.WireframeGeometry(geometry),
    new THREE.LineBasicMaterial({
      color: isSelected ? "#f59e0b" : toThreeColor(materialInfo?.color ?? [0.9, 0.8, 0.5]),
      transparent: true,
      opacity: 0.92
    })
  );
  const ghost = new THREE.Mesh(
    geometry,
    new THREE.MeshBasicMaterial({
      color: toThreeColor(materialInfo?.color ?? [0.9, 0.8, 0.5]),
      transparent: true,
      opacity: 0.08
    })
  );
  const anchor = new THREE.Mesh(
    new THREE.SphereGeometry(24, 18, 18),
    new THREE.MeshBasicMaterial({
      color: isSelected ? "#f97316" : "#cbd5e1"
    })
  );
  const group = new THREE.Group();
  const center = estimateObjectCenter(object);
  group.position.fromArray(center);
  group.add(ghost, wireframe, anchor);
  return group;
}

function decorateSelection(
  mesh: THREE.Mesh,
  geometry: THREE.BufferGeometry,
  isSelected: boolean
) {
  if (isSelected) {
    const highlight = new THREE.LineSegments(
      new THREE.EdgesGeometry(geometry),
      new THREE.LineBasicMaterial({ color: "#f97316" })
    );
    mesh.add(highlight);
  }

  return mesh;
}

function createSurfaceMaterial(
  materialInfo: SceneMaterial | undefined,
  isSelected: boolean,
  isProxy: boolean
) {
  const baseColor = toThreeColor(materialInfo?.color ?? [0.88, 0.89, 0.95]);
  const reflectivity = Math.min(1, Math.max(0, toNumber(materialInfo?.reflectivity, 0.18)));
  const refractivity = Math.max(0, toNumber(materialInfo?.refractivity));
  const emitsLight = materialEmitsLight(materialInfo);
  const opacity = emitsLight ? 0.98 : refractivity > 0 ? 0.42 : 0.9;

  return new THREE.MeshStandardMaterial({
    color: baseColor,
    transparent: opacity < 0.99,
    opacity,
    roughness: Math.max(0.08, 0.88 - reflectivity * 0.7),
    metalness: Math.min(0.95, reflectivity),
    emissive: emitsLight ? baseColor.clone() : new THREE.Color(isSelected ? "#9a3412" : "#000000"),
    emissiveIntensity: emitsLight ? 1.15 : isSelected ? 0.18 : 0,
    wireframe: isProxy
  });
}

function toThreeColor(color: number[]) {
  const [r, g, b] = color;
  return new THREE.Color(
    Math.min(1, Math.max(0, toNumber(r, 1))),
    Math.min(1, Math.max(0, toNumber(g, 1))),
    Math.min(1, Math.max(0, toNumber(b, 1)))
  );
}

function clearGroup(group: THREE.Group) {
  while (group.children.length > 0) {
    const child = group.children[0];
    group.remove(child);
    disposeObject(child);
  }
}

function disposeObject(object: THREE.Object3D) {
  object.traverse((entry) => {
    const mesh = entry as THREE.Mesh;
    if (mesh.geometry) {
      mesh.geometry.dispose();
    }

    if (Array.isArray(mesh.material)) {
      mesh.material.forEach((material) => material.dispose());
    } else if (mesh.material) {
      mesh.material.dispose();
    }
  });
}
