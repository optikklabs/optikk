package k8sapply

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// FieldManager identifies the CLI as the owner of applied fields (SSA).
const FieldManager = "optikk-cli"

// Applier server-side-applies rendered objects, ordering CRDs and namespaces
// ahead of the resources that depend on them.
type Applier struct {
	dyn    dynamic.Interface
	mapper *restmapper.DeferredDiscoveryRESTMapper
	disco  discovery.CachedDiscoveryInterface
}

// NewApplier builds an applier from a REST config.
func NewApplier(cfg *rest.Config) (*Applier, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	cached := memory.NewMemCacheClient(dc)
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Applier{
		dyn:    dyn,
		mapper: restmapper.NewDeferredDiscoveryRESTMapper(cached),
		disco:  cached,
	}, nil
}

// Apply applies every object. Namespaces and CRDs go first; the remainder is
// retried so CRs land after their CRDs register in discovery.
func (a *Applier) Apply(ctx context.Context, objs []*unstructured.Unstructured) error {
	var first, rest []*unstructured.Unstructured
	for _, o := range objs {
		if isPriority(o) {
			first = append(first, o)
		} else {
			rest = append(rest, o)
		}
	}

	for _, o := range first {
		if err := a.applyOne(ctx, o); err != nil {
			return err
		}
	}
	// New CRDs/namespaces changed discovery — force a fresh mapping.
	a.reset()
	return a.applyWithRetry(ctx, rest)
}

// applyWithRetry retries objects whose kind isn't mapped yet (CRD still
// registering), resetting discovery between passes.
func (a *Applier) applyWithRetry(ctx context.Context, objs []*unstructured.Unstructured) error {
	pending := objs
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		var next []*unstructured.Unstructured
		for _, o := range pending {
			if err := a.applyOne(ctx, o); err != nil {
				if meta.IsNoMatchError(err) {
					next = append(next, o)
					lastErr = err
					continue
				}
				return fmt.Errorf("apply %s/%s: %w", o.GetKind(), o.GetName(), err)
			}
		}
		if len(next) == 0 {
			return nil
		}
		pending = next
		a.reset()
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("apply timed out waiting for CRDs to register: %w", lastErr)
}

func (a *Applier) applyOne(ctx context.Context, o *unstructured.Unstructured) error {
	gvk := o.GroupVersionKind()
	mapping, err := a.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}
	ri := a.resourceFor(mapping, o)
	_, err = ri.Apply(ctx, o.GetName(), o, metav1.ApplyOptions{
		FieldManager: FieldManager,
		Force:        true,
	})
	return err
}

// Delete removes objects, ignoring those already gone. CRDs/namespaces are
// deleted last so dependent resources go first.
func (a *Applier) Delete(ctx context.Context, objs []*unstructured.Unstructured) error {
	var last []*unstructured.Unstructured
	for _, o := range objs {
		if isPriority(o) {
			last = append(last, o)
			continue
		}
		if err := a.deleteOne(ctx, o); err != nil {
			return err
		}
	}
	for _, o := range last {
		if err := a.deleteOne(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

func (a *Applier) deleteOne(ctx context.Context, o *unstructured.Unstructured) error {
	gvk := o.GroupVersionKind()
	mapping, err := a.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		if meta.IsNoMatchError(err) {
			return nil // kind's CRD already gone
		}
		return err
	}
	err = a.resourceFor(mapping, o).Delete(ctx, o.GetName(), metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (a *Applier) resourceFor(mapping *meta.RESTMapping, o *unstructured.Unstructured) dynamic.ResourceInterface {
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		return a.dyn.Resource(mapping.Resource).Namespace(o.GetNamespace())
	}
	return a.dyn.Resource(mapping.Resource)
}

func (a *Applier) reset() {
	a.disco.Invalidate()
	a.mapper.Reset()
}

// isPriority marks objects that must exist before others can apply.
func isPriority(o *unstructured.Unstructured) bool {
	switch o.GroupVersionKind() {
	case schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}:
		return true
	case schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}:
		return true
	}
	return false
}
