/*
Copyright 2023 nineinfra.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/nineinfra/metastore-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMetastoreClusters implements MetastoreClusterInterface
type FakeMetastoreClusters struct {
	Fake *FakeMetastoreV1alpha1
	ns   string
}

var metastoreclustersResource = v1alpha1.SchemeGroupVersion.WithResource("metastoreclusters")

var metastoreclustersKind = v1alpha1.SchemeGroupVersion.WithKind("MetastoreCluster")

// Get takes name of the metastoreCluster, and returns the corresponding metastoreCluster object, and an error if there is any.
func (c *FakeMetastoreClusters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MetastoreCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(metastoreclustersResource, c.ns, name), &v1alpha1.MetastoreCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetastoreCluster), err
}

// List takes label and field selectors, and returns the list of MetastoreClusters that match those selectors.
func (c *FakeMetastoreClusters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MetastoreClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(metastoreclustersResource, metastoreclustersKind, c.ns, opts), &v1alpha1.MetastoreClusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MetastoreClusterList{ListMeta: obj.(*v1alpha1.MetastoreClusterList).ListMeta}
	for _, item := range obj.(*v1alpha1.MetastoreClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested metastoreClusters.
func (c *FakeMetastoreClusters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(metastoreclustersResource, c.ns, opts))

}

// Create takes the representation of a metastoreCluster and creates it.  Returns the server's representation of the metastoreCluster, and an error, if there is any.
func (c *FakeMetastoreClusters) Create(ctx context.Context, metastoreCluster *v1alpha1.MetastoreCluster, opts v1.CreateOptions) (result *v1alpha1.MetastoreCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(metastoreclustersResource, c.ns, metastoreCluster), &v1alpha1.MetastoreCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetastoreCluster), err
}

// Update takes the representation of a metastoreCluster and updates it. Returns the server's representation of the metastoreCluster, and an error, if there is any.
func (c *FakeMetastoreClusters) Update(ctx context.Context, metastoreCluster *v1alpha1.MetastoreCluster, opts v1.UpdateOptions) (result *v1alpha1.MetastoreCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(metastoreclustersResource, c.ns, metastoreCluster), &v1alpha1.MetastoreCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetastoreCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMetastoreClusters) UpdateStatus(ctx context.Context, metastoreCluster *v1alpha1.MetastoreCluster, opts v1.UpdateOptions) (*v1alpha1.MetastoreCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(metastoreclustersResource, "status", c.ns, metastoreCluster), &v1alpha1.MetastoreCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetastoreCluster), err
}

// Delete takes name of the metastoreCluster and deletes it. Returns an error if one occurs.
func (c *FakeMetastoreClusters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(metastoreclustersResource, c.ns, name, opts), &v1alpha1.MetastoreCluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMetastoreClusters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(metastoreclustersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.MetastoreClusterList{})
	return err
}

// Patch applies the patch and returns the patched metastoreCluster.
func (c *FakeMetastoreClusters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MetastoreCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(metastoreclustersResource, c.ns, name, pt, data, subresources...), &v1alpha1.MetastoreCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MetastoreCluster), err
}
