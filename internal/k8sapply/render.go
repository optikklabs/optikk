// Package k8sapply renders a kustomize overlay and server-side-applies it.
package k8sapply

import (
	"bytes"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// Render builds a kustomize overlay from disk into a list of objects.
func Render(overlayDir string) ([]*unstructured.Unstructured, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := k.Run(filesys.MakeFsOnDisk(), overlayDir)
	if err != nil {
		return nil, fmt.Errorf("kustomize build %s: %w", overlayDir, err)
	}
	out, err := resMap.AsYaml()
	if err != nil {
		return nil, err
	}
	return decode(out)
}

// decode splits a multi-doc YAML stream into unstructured objects.
func decode(manifest []byte) ([]*unstructured.Unstructured, error) {
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 4096)
	var objs []*unstructured.Unstructured
	for {
		obj := &unstructured.Unstructured{}
		if err := dec.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode manifest: %w", err)
		}
		if len(obj.Object) == 0 {
			continue // skip empty documents
		}
		objs = append(objs, obj)
	}
	return objs, nil
}
