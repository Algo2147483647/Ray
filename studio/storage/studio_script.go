package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Algo2147483647/ray/studio/schema"
)

func ReadStudioScriptFiles(paths []string) (*schema.StudioScript, error) {
	if len(paths) == 0 {
		return nil, errors.New("no script files provided")
	}

	merged := &schema.StudioScript{}
	for _, path := range paths {
		script, err := readStudioScriptFile(path)
		if err != nil {
			return nil, err
		}
		if err := mergeStudioScripts(merged, script, path); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func readStudioScriptFile(path string) (*schema.StudioScript, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve script path %q: %w", path, err)
	}
	return readStudioScriptFileRecursive(filepath.Clean(absolute), map[string]bool{})
}

func readStudioScriptFileRecursive(path string, stack map[string]bool) (*schema.StudioScript, error) {
	if stack[path] {
		return nil, fmt.Errorf("include cycle detected at %q", path)
	}
	stack[path] = true
	defer delete(stack, path)

	script, err := readStudioScriptFileRaw(path)
	if err != nil {
		return nil, err
	}

	merged := &schema.StudioScript{}
	for _, include := range script.Includes {
		includePath := include
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(filepath.Dir(path), includePath)
		}
		includePath, err = filepath.Abs(includePath)
		if err != nil {
			return nil, fmt.Errorf("resolve script path %q: %w", include, err)
		}
		included, err := readStudioScriptFileRecursive(filepath.Clean(includePath), stack)
		if err != nil {
			return nil, err
		}
		if err := mergeStudioScripts(merged, included, includePath); err != nil {
			return nil, err
		}
	}

	script.Includes = nil
	if err := mergeStudioScripts(merged, script, path); err != nil {
		return nil, err
	}
	return merged, nil
}

func readStudioScriptFileRaw(path string) (*schema.StudioScript, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open script %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", path, err)
	}

	var script schema.StudioScript
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("parse script %q: %w", path, err)
	}
	return &script, nil
}

func mergeStudioScripts(dst, src *schema.StudioScript, source string) error {
	if dst == nil || src == nil {
		return errors.New("cannot merge nil script")
	}
	incomingObjects := cloneStudioObjectMaps(src.Objects)
	var err error
	dst.Objects, err = normalizeStudioObjectTree(dst.Objects, source)
	if err != nil {
		return err
	}
	incomingObjects, err = normalizeStudioObjectTree(incomingObjects, source)
	if err != nil {
		return err
	}
	if err := bindUnparentedStudioObjects(&dst.Objects, &incomingObjects, source); err != nil {
		return err
	}
	if err := validateStudioObjectParents(dst.Objects, incomingObjects, source); err != nil {
		return err
	}
	if err := mergeStudioMedia(dst, src, source); err != nil {
		return err
	}
	if err := appendUniqueStudioIDMaps(&dst.Materials, src.Materials, "material", source); err != nil {
		return err
	}
	if err := appendOrMergeStudioObjects(&dst.Objects, incomingObjects, source); err != nil {
		return err
	}
	if err := appendUniqueStudioCameras(&dst.Cameras, src.Cameras, source); err != nil {
		return err
	}
	if err := appendUniqueStudioFilms(&dst.Films, src.Films, source); err != nil {
		return err
	}
	if src.Dimension > 0 {
		if dst.Dimension > 0 && dst.Dimension != src.Dimension {
			return fmt.Errorf("scene dimension %d from %s conflicts with dimension %d", src.Dimension, source, dst.Dimension)
		}
		dst.Dimension = src.Dimension
	}
	if src.Render.LegacyDimension > 0 && dst.Render.LegacyDimension > 0 &&
		src.Render.LegacyDimension != dst.Render.LegacyDimension {
		return fmt.Errorf(
			"legacy render dimension %d from %s conflicts with dimension %d",
			src.Render.LegacyDimension, source, dst.Render.LegacyDimension,
		)
	}
	dst.Render = schema.MergeRenderScripts(dst.Render, src.Render)
	if len(src.Geometry) > 0 {
		dst.Geometry = cloneMap(src.Geometry)
	}
	dst.Renders = append(dst.Renders, src.Renders...)
	return nil
}

func appendUniqueStudioFilms(dst *[]schema.StudioFilmScript, src []schema.StudioFilmScript, source string) error {
	ids := map[string]bool{}
	for _, film := range *dst {
		if film.ID != "" {
			ids[film.ID] = true
		}
	}
	for _, film := range src {
		if film.ID != "" && ids[film.ID] {
			return fmt.Errorf("duplicate film id %q from %s", film.ID, source)
		}
		if film.ID != "" {
			ids[film.ID] = true
		}
		film.Shape = append([]int(nil), film.Shape...)
		film.PixelWindows = cloneStudioPixelWindows(film.PixelWindows)
		*dst = append(*dst, film)
	}
	return nil
}

func mergeStudioMedia(dst, src *schema.StudioScript, source string) error {
	if len(src.Media) == 0 {
		return nil
	}
	if dst.Media == nil {
		dst.Media = map[string]map[string]interface{}{}
	}
	for id, medium := range src.Media {
		if _, exists := dst.Media[id]; exists {
			return fmt.Errorf("duplicate medium id %q while merging %s", id, source)
		}
		dst.Media[id] = cloneMap(medium)
	}
	return nil
}

func appendUniqueStudioIDMaps(dst *[]map[string]interface{}, src []map[string]interface{}, label, source string) error {
	ids := map[string]bool{}
	for _, item := range *dst {
		if id, ok := stringField(item, "id"); ok {
			ids[id] = true
		}
	}
	for _, item := range src {
		id, ok := stringField(item, "id")
		if !ok {
			*dst = append(*dst, cloneMap(item))
			continue
		}
		if ids[id] {
			return fmt.Errorf("duplicate %s id %q while merging %s", label, id, source)
		}
		ids[id] = true
		*dst = append(*dst, cloneMap(item))
	}
	return nil
}

func appendOrMergeStudioObjects(dst *[]map[string]interface{}, src []map[string]interface{}, source string) error {
	ids := map[string]int{}
	for index, item := range *dst {
		if id, ok := stringField(item, "id"); ok {
			ids[id] = index
		}
	}
	for _, item := range src {
		id, ok := stringField(item, "id")
		if !ok {
			*dst = append(*dst, cloneMap(item))
			continue
		}
		if existingIndex, exists := ids[id]; exists {
			merged, err := mergeStudioObject((*dst)[existingIndex], item, source)
			if err != nil {
				return err
			}
			(*dst)[existingIndex] = merged
			continue
		}
		ids[id] = len(*dst)
		*dst = append(*dst, cloneMap(item))
	}
	return nil
}

func mergeStudioObject(base, override map[string]interface{}, source string) (map[string]interface{}, error) {
	baseShape, _ := stringField(base, "shape")
	overrideShape, _ := stringField(override, "shape")
	baseShape = strings.ToLower(baseShape)
	overrideShape = strings.ToLower(overrideShape)
	if !mergeableContainerShape(baseShape) || !mergeableContainerShape(overrideShape) || baseShape != overrideShape {
		id, _ := stringField(base, "id")
		return nil, fmt.Errorf("duplicate object id %q while merging %s", id, source)
	}

	merged := cloneMap(base)
	overrideClone := cloneMap(override)
	for key, value := range overrideClone {
		if key == "objects" {
			continue
		}
		existing, exists := merged[key]
		if !exists {
			merged[key] = value
			continue
		}
		// Underscore-prefixed fields are authoring metadata, not render
		// parameters. Keep the first description when fragments differ.
		if strings.HasPrefix(key, "_") || reflect.DeepEqual(existing, value) {
			continue
		}
		id, _ := stringField(base, "id")
		return nil, fmt.Errorf(
			"object id %q has conflicting field %q while merging %s: %s != %s",
			id, key, source, formatStudioValue(existing), formatStudioValue(value),
		)
	}

	switch baseShape {
	case "group":
		if objects, ok, err := mergeOptionalStudioObjectLists(merged["objects"], override["objects"], source); err != nil {
			return nil, err
		} else if ok {
			merged["objects"] = objects
		}
	case "array":
		if objects, ok, err := mergeOptionalStudioArrayObjects(merged["objects"], override["objects"], source); err != nil {
			return nil, err
		} else if ok {
			merged["objects"] = objects
		}
	}
	return merged, nil
}

type studioObjectLocation struct {
	parent     string
	comparable bool
	unbound    bool
}

type nestedStudioObject struct {
	object map[string]interface{}
	parent string
}

func normalizeStudioObjectTree(objects []map[string]interface{}, source string) ([]map[string]interface{}, error) {
	result := []map[string]interface{}{}
	for _, object := range objects {
		normalized, err := normalizeStudioObjectChildren(object, source)
		if err != nil {
			return nil, err
		}
		if err := appendOrMergeStudioObjects(&result, []map[string]interface{}{normalized}, source); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func normalizeStudioObjectChildren(object map[string]interface{}, source string) (map[string]interface{}, error) {
	normalized := cloneMap(object)
	shape, _ := stringField(normalized, "shape")
	switch strings.ToLower(shape) {
	case "group":
		raw, exists := normalized["objects"]
		if !exists || raw == nil {
			return normalized, nil
		}
		items, err := objectListRaw(raw, "objects")
		if err != nil {
			return nil, err
		}
		children, err := studioObjectMapsFromInterfaces(items, "objects")
		if err != nil {
			return nil, err
		}
		children, err = normalizeStudioObjectTree(children, source)
		if err != nil {
			return nil, err
		}
		normalized["objects"] = studioObjectInterfaces(children)
	case "array":
		raw, exists := normalized["objects"]
		if !exists || raw == nil {
			return normalized, nil
		}
		cells, err := objectMapRaw(raw, "objects")
		if err != nil {
			return nil, err
		}
		normalizedCells := map[string]interface{}{}
		for cell, cellRaw := range cells {
			items, err := objectListRaw(cellRaw, "objects."+cell)
			if err != nil {
				return nil, err
			}
			children, err := studioObjectMapsFromInterfaces(items, "objects."+cell)
			if err != nil {
				return nil, err
			}
			children, err = normalizeStudioObjectTree(children, source)
			if err != nil {
				return nil, err
			}
			normalizedCells[cell] = studioObjectInterfaces(children)
		}
		normalized["objects"] = normalizedCells
	}
	return normalized, nil
}

func studioObjectMapsFromInterfaces(items []interface{}, field string) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %q: expected object, got %T", field, item)
		}
		result = append(result, object)
	}
	return result, nil
}

func studioObjectInterfaces(objects []map[string]interface{}) []interface{} {
	result := make([]interface{}, len(objects))
	for index, object := range objects {
		result[index] = object
	}
	return result
}

func bindUnparentedStudioObjects(base, incoming *[]map[string]interface{}, source string) error {
	if err := bindUnparentedStudioObjectsWithin(base, source); err != nil {
		return err
	}
	if err := bindUnparentedStudioObjectsWithin(incoming, source); err != nil {
		return err
	}

	baseRoots := topLevelStudioObjectIndices(*base)
	incomingRoots := topLevelStudioObjectIndices(*incoming)
	baseNested, err := nestedStudioObjects(*base, source)
	if err != nil {
		return err
	}
	incomingNested, err := nestedStudioObjects(*incoming, source)
	if err != nil {
		return err
	}

	removeBase := map[int]bool{}
	removeIncoming := map[int]bool{}
	for id, baseIndex := range baseRoots {
		nested, exists := incomingNested[id]
		if !exists {
			continue
		}
		merged, err := mergeStudioObject((*base)[baseIndex], nested.object, source)
		if err != nil {
			return err
		}
		replaceStudioObject(nested.object, merged)
		removeBase[baseIndex] = true
	}
	for id, incomingIndex := range incomingRoots {
		nested, exists := baseNested[id]
		if !exists {
			continue
		}
		merged, err := mergeStudioObject(nested.object, (*incoming)[incomingIndex], source)
		if err != nil {
			return err
		}
		replaceStudioObject(nested.object, merged)
		removeIncoming[incomingIndex] = true
	}
	*base = removeStudioObjectIndices(*base, removeBase)
	*incoming = removeStudioObjectIndices(*incoming, removeIncoming)
	return nil
}

func bindUnparentedStudioObjectsWithin(objects *[]map[string]interface{}, source string) error {
	roots := topLevelStudioObjectIndices(*objects)
	nested, err := nestedStudioObjects(*objects, source)
	if err != nil {
		return err
	}
	remove := map[int]bool{}
	for id, rootIndex := range roots {
		nestedObject, exists := nested[id]
		if !exists {
			continue
		}
		if strings.HasPrefix(nestedObject.parent, "$/"+id+"/") || nestedObject.parent == "$/"+id {
			return fmt.Errorf("object id %q cannot bind to its own descendant in %s", id, source)
		}
		merged, err := mergeStudioObject((*objects)[rootIndex], nestedObject.object, source)
		if err != nil {
			return err
		}
		replaceStudioObject(nestedObject.object, merged)
		remove[rootIndex] = true
	}
	*objects = removeStudioObjectIndices(*objects, remove)
	return nil
}

func topLevelStudioObjectIndices(objects []map[string]interface{}) map[string]int {
	result := map[string]int{}
	for index, object := range objects {
		if id, ok := stringField(object, "id"); ok {
			result[id] = index
		}
	}
	return result
}

func nestedStudioObjects(objects []map[string]interface{}, source string) (map[string]nestedStudioObject, error) {
	result := map[string]nestedStudioObject{}
	for index, object := range objects {
		id, hasID := stringField(object, "id")
		segment := id
		if !hasID {
			segment = fmt.Sprintf("<anonymous:%d>", index)
		}
		if err := collectNestedStudioObjects(object, "$/"+segment, result, source); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func collectNestedStudioObjects(
	object map[string]interface{},
	objectPath string,
	result map[string]nestedStudioObject,
	source string,
) error {
	shape, _ := stringField(object, "shape")
	switch strings.ToLower(shape) {
	case "group":
		raw, exists := object["objects"]
		if !exists || raw == nil {
			return nil
		}
		items, err := objectListRaw(raw, "objects")
		if err != nil {
			return err
		}
		for index, item := range items {
			child, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("field %q: expected object, got %T", "objects", item)
			}
			if err := addNestedStudioObject(child, index, objectPath, result, source); err != nil {
				return err
			}
		}
	case "array":
		raw, exists := object["objects"]
		if !exists || raw == nil {
			return nil
		}
		cells, err := objectMapRaw(raw, "objects")
		if err != nil {
			return err
		}
		for cell, cellRaw := range cells {
			items, err := objectListRaw(cellRaw, "objects."+cell)
			if err != nil {
				return err
			}
			for index, item := range items {
				child, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("field %q: expected object, got %T", "objects."+cell, item)
				}
				if err := addNestedStudioObject(child, index, objectPath+"["+cell+"]", result, source); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func addNestedStudioObject(
	object map[string]interface{},
	index int,
	parent string,
	result map[string]nestedStudioObject,
	source string,
) error {
	id, hasID := stringField(object, "id")
	segment := id
	if !hasID {
		segment = fmt.Sprintf("<anonymous:%d>", index)
	} else if existing, exists := result[id]; exists {
		if existing.parent != parent {
			return fmt.Errorf("object id %q has conflicting parents in %s: %q != %q", id, source, existing.parent, parent)
		}
		return fmt.Errorf("duplicate object id %q under parent %q in %s", id, parent, source)
	} else {
		result[id] = nestedStudioObject{object: object, parent: parent}
	}
	return collectNestedStudioObjects(object, parent+"/"+segment, result, source)
}

func replaceStudioObject(target, replacement map[string]interface{}) {
	for key := range target {
		delete(target, key)
	}
	for key, value := range replacement {
		target[key] = value
	}
}

func removeStudioObjectIndices(objects []map[string]interface{}, remove map[int]bool) []map[string]interface{} {
	if len(remove) == 0 {
		return objects
	}
	result := make([]map[string]interface{}, 0, len(objects)-len(remove))
	for index, object := range objects {
		if !remove[index] {
			result = append(result, object)
		}
	}
	return result
}

func validateStudioObjectParents(base, incoming []map[string]interface{}, source string) error {
	baseLocations := map[string]studioObjectLocation{}
	if err := collectStudioObjectLocations(base, "$", false, baseLocations, "merged scripts"); err != nil {
		return err
	}
	incomingLocations := map[string]studioObjectLocation{}
	if err := collectStudioObjectLocations(incoming, "$", false, incomingLocations, source); err != nil {
		return err
	}
	for id, incomingLocation := range incomingLocations {
		baseLocation, exists := baseLocations[id]
		if !exists {
			continue
		}
		if studioObjectParentsConflict(baseLocation, incomingLocation) {
			return fmt.Errorf(
				"object id %q has conflicting parents while merging %s: %q != %q",
				id, source, baseLocation.parent, incomingLocation.parent,
			)
		}
	}
	return nil
}

func collectStudioObjectLocations(
	objects []map[string]interface{},
	parent string,
	parentComparable bool,
	locations map[string]studioObjectLocation,
	source string,
) error {
	for index, object := range objects {
		if err := collectStudioObjectLocation(object, index, parent, parentComparable, locations, source); err != nil {
			return err
		}
	}
	return nil
}

func collectStudioObjectLocation(
	object map[string]interface{},
	index int,
	parent string,
	parentComparable bool,
	locations map[string]studioObjectLocation,
	source string,
) error {
	id, hasID := stringField(object, "id")
	if hasID {
		location := studioObjectLocation{parent: parent, comparable: parentComparable, unbound: parent == "$"}
		if existing, exists := locations[id]; exists &&
			studioObjectParentsConflict(existing, location) {
			return fmt.Errorf(
				"object id %q has conflicting parents in %s: %q != %q",
				id, source, existing.parent, location.parent,
			)
		}
		locations[id] = location
	}

	segment := id
	childParentsComparable := hasID
	if !hasID {
		segment = fmt.Sprintf("<anonymous:%d>", index)
	}
	objectPath := parent + "/" + segment
	shape, _ := stringField(object, "shape")
	switch strings.ToLower(shape) {
	case "group":
		raw, exists := object["objects"]
		if !exists || raw == nil {
			return nil
		}
		items, err := objectListRaw(raw, "objects")
		if err != nil {
			return err
		}
		children := make([]map[string]interface{}, 0, len(items))
		for _, item := range items {
			child, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("field %q: expected object, got %T", "objects", item)
			}
			children = append(children, child)
		}
		return collectStudioObjectLocations(children, objectPath, childParentsComparable, locations, source)
	case "array":
		raw, exists := object["objects"]
		if !exists || raw == nil {
			return nil
		}
		cells, err := objectMapRaw(raw, "objects")
		if err != nil {
			return err
		}
		for cell, cellRaw := range cells {
			items, err := objectListRaw(cellRaw, "objects."+cell)
			if err != nil {
				return err
			}
			for childIndex, item := range items {
				child, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("field %q: expected object, got %T", "objects."+cell, item)
				}
				if err := collectStudioObjectLocation(
					child, childIndex, objectPath+"["+cell+"]", childParentsComparable, locations, source,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func studioObjectParentsConflict(first, second studioObjectLocation) bool {
	if first.unbound || second.unbound {
		return false
	}
	return !first.comparable || !second.comparable || first.parent != second.parent
}

func formatStudioValue(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}
	return string(data)
}

func mergeableContainerShape(shape string) bool {
	return shape == "group" || shape == "array"
}

func mergeStudioObjectLists(baseRaw, overrideRaw interface{}, source string) ([]interface{}, error) {
	baseItems, err := objectListRaw(baseRaw, "objects")
	if err != nil {
		return nil, err
	}
	overrideItems, err := objectListRaw(overrideRaw, "objects")
	if err != nil {
		return nil, err
	}

	merged := cloneInterfaceSlice(baseItems)
	ids := map[string]int{}
	for index, item := range merged {
		object, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := stringField(object, "id"); ok {
			ids[id] = index
		}
	}
	for _, item := range overrideItems {
		object, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %q: expected object, got %T", "objects", item)
		}
		id, hasID := stringField(object, "id")
		if !hasID {
			merged = append(merged, cloneInterfaceValue(item))
			continue
		}
		if existingIndex, exists := ids[id]; exists {
			existing, ok := merged[existingIndex].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("duplicate object id %q while merging %s", id, source)
			}
			nested, err := mergeStudioObject(existing, object, source)
			if err != nil {
				return nil, err
			}
			merged[existingIndex] = nested
			continue
		}
		ids[id] = len(merged)
		merged = append(merged, cloneInterfaceValue(item))
	}
	return merged, nil
}

func mergeOptionalStudioObjectLists(baseRaw, overrideRaw interface{}, source string) ([]interface{}, bool, error) {
	if baseRaw == nil && overrideRaw == nil {
		return nil, false, nil
	}
	if baseRaw == nil {
		overrideItems, err := objectListRaw(overrideRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneInterfaceSlice(overrideItems), true, nil
	}
	if overrideRaw == nil {
		baseItems, err := objectListRaw(baseRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneInterfaceSlice(baseItems), true, nil
	}
	merged, err := mergeStudioObjectLists(baseRaw, overrideRaw, source)
	return merged, err == nil, err
}

func mergeStudioArrayObjects(baseRaw, overrideRaw interface{}, source string) (map[string]interface{}, error) {
	baseMap, err := objectMapRaw(baseRaw, "objects")
	if err != nil {
		return nil, err
	}
	overrideMap, err := objectMapRaw(overrideRaw, "objects")
	if err != nil {
		return nil, err
	}
	merged := cloneStringInterfaceMap(baseMap)
	for cell, overrideItems := range overrideMap {
		baseItems, exists := merged[cell]
		if !exists {
			merged[cell] = cloneInterfaceValue(overrideItems)
			continue
		}
		items, err := mergeStudioObjectLists(baseItems, overrideItems, source)
		if err != nil {
			return nil, err
		}
		merged[cell] = items
	}
	return merged, nil
}

func mergeOptionalStudioArrayObjects(baseRaw, overrideRaw interface{}, source string) (map[string]interface{}, bool, error) {
	if baseRaw == nil && overrideRaw == nil {
		return nil, false, nil
	}
	if baseRaw == nil {
		overrideMap, err := objectMapRaw(overrideRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneStringInterfaceMap(overrideMap), true, nil
	}
	if overrideRaw == nil {
		baseMap, err := objectMapRaw(baseRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneStringInterfaceMap(baseMap), true, nil
	}
	merged, err := mergeStudioArrayObjects(baseRaw, overrideRaw, source)
	return merged, err == nil, err
}

func objectListRaw(raw interface{}, key string) ([]interface{}, error) {
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array, got %T", key, raw)
	}
	return items, nil
}

func objectMapRaw(raw interface{}, key string) (map[string]interface{}, error) {
	items, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected object, got %T", key, raw)
	}
	return items, nil
}

func cloneStringInterfaceMap(value map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(value))
	for key, item := range value {
		result[key] = cloneInterfaceValue(item)
	}
	return result
}

func cloneInterfaceSlice(value []interface{}) []interface{} {
	result := make([]interface{}, len(value))
	for i, item := range value {
		result[i] = cloneInterfaceValue(item)
	}
	return result
}

func cloneInterfaceValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return cloneMap(v)
	case []interface{}:
		return cloneInterfaceSlice(v)
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = cloneMap(item)
		}
		return result
	case []float64:
		return append([]float64(nil), v...)
	case []string:
		return append([]string(nil), v...)
	default:
		return value
	}
}

func cloneMap(value map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(value))
	for key, item := range value {
		result[key] = cloneInterfaceValue(item)
	}
	return result
}

func cloneStudioObjectMaps(value []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, len(value))
	for index, object := range value {
		result[index] = cloneMap(object)
	}
	return result
}

func stringField(object map[string]interface{}, key string) (string, bool) {
	raw, ok := object[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	return value, ok && value != ""
}

func cloneStudioCamera(def schema.StudioCameraScript) schema.StudioCameraScript {
	camera := def
	camera.Position = append([]float64(nil), def.Position...)
	camera.LookAt = append([]float64(nil), def.LookAt...)
	camera.Direction = append([]float64(nil), def.Direction...)
	camera.Up = append([]float64(nil), def.Up...)
	camera.FieldOfViews = append([]float64(nil), def.FieldOfViews...)
	if len(def.Coordinates) > 0 {
		camera.Coordinates = make([][]float64, len(def.Coordinates))
		for i, coordinate := range def.Coordinates {
			camera.Coordinates[i] = append([]float64(nil), coordinate...)
		}
	}
	return camera
}

func appendUniqueStudioCameras(dst *[]schema.StudioCameraScript, src []schema.StudioCameraScript, source string) error {
	ids := map[string]bool{}
	for _, camera := range *dst {
		if camera.ID != "" {
			ids[camera.ID] = true
		}
	}
	for _, camera := range src {
		if camera.ID != "" {
			if ids[camera.ID] {
				return fmt.Errorf("duplicate camera id %q while merging %s", camera.ID, source)
			}
			ids[camera.ID] = true
		}
		*dst = append(*dst, cloneStudioCamera(camera))
	}
	return nil
}

func cloneStudioPixelWindows(windows []schema.PixelWindowScript) []schema.PixelWindowScript {
	if len(windows) == 0 {
		return nil
	}
	cloned := make([]schema.PixelWindowScript, len(windows))
	for i, window := range windows {
		cloned[i] = schema.PixelWindowScript{
			Min: append([]int(nil), window.Min...),
			Max: append([]int(nil), window.Max...),
		}
	}
	return cloned
}
