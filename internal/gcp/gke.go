// Package gcp provisions GKE clusters and GCS buckets for the gcp target.
package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"golang.org/x/oauth2/google"
	"k8s.io/client-go/rest"
)

// cloudPlatformScope is the OAuth scope for GKE control-plane access.
const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// ClusterSpec describes the GKE cluster to create.
type ClusterSpec struct {
	Project     string
	Region      string
	Name        string
	Nodes       int32
	MinNodes    int32
	MaxNodes    int32
	MachineType string
}

func (s ClusterSpec) parent() string {
	return fmt.Sprintf("projects/%s/locations/%s", s.Project, s.Region)
}

func (s ClusterSpec) self() string {
	return fmt.Sprintf("%s/clusters/%s", s.parent(), s.Name)
}

// CreateGKE creates a regional autoscaling cluster and waits for it to finish.
func CreateGKE(ctx context.Context, s ClusterSpec) error {
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	req := &containerpb.CreateClusterRequest{
		Parent: s.parent(),
		Cluster: &containerpb.Cluster{
			Name: s.Name,
			NodePools: []*containerpb.NodePool{{
				Name:             "default",
				InitialNodeCount: s.Nodes,
				Config:           &containerpb.NodeConfig{MachineType: s.MachineType},
				Autoscaling: &containerpb.NodePoolAutoscaling{
					Enabled:      true,
					MinNodeCount: s.MinNodes,
					MaxNodeCount: s.MaxNodes,
				},
			}},
		},
	}
	op, err := c.CreateCluster(ctx, req)
	if err != nil {
		return fmt.Errorf("create cluster: %w", err)
	}
	return waitOp(ctx, c, s.parent(), op)
}

// DeleteGKE deletes the cluster and waits for completion.
func DeleteGKE(ctx context.Context, s ClusterSpec) error {
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	op, err := c.DeleteCluster(ctx, &containerpb.DeleteClusterRequest{Name: s.self()})
	if err != nil {
		return fmt.Errorf("delete cluster: %w", err)
	}
	return waitOp(ctx, c, s.parent(), op)
}

// RESTConfig builds a client-go config for the cluster's API server using
// application-default credentials for the bearer token.
func RESTConfig(ctx context.Context, s ClusterSpec) (*rest.Config, error) {
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	cluster, err := c.GetCluster(ctx, &containerpb.GetClusterRequest{Name: s.self()})
	if err != nil {
		return nil, err
	}
	ca, err := base64.StdEncoding.DecodeString(cluster.GetMasterAuth().GetClusterCaCertificate())
	if err != nil {
		return nil, fmt.Errorf("decode cluster CA: %w", err)
	}
	ts, err := google.DefaultTokenSource(ctx, cloudPlatformScope)
	if err != nil {
		return nil, err
	}
	tok, err := ts.Token()
	if err != nil {
		return nil, err
	}
	return &rest.Config{
		Host:            "https://" + cluster.GetEndpoint(),
		BearerToken:     tok.AccessToken,
		TLSClientConfig: rest.TLSClientConfig{CAData: ca},
	}, nil
}

// waitOp polls a cluster operation until it reports DONE.
func waitOp(ctx context.Context, c *container.ClusterManagerClient, parent string, op *containerpb.Operation) error {
	name := fmt.Sprintf("%s/operations/%s", parent, op.GetName())
	for {
		got, err := c.GetOperation(ctx, &containerpb.GetOperationRequest{Name: name})
		if err != nil {
			return err
		}
		if got.GetStatus() == containerpb.Operation_DONE {
			if e := got.GetError(); e != nil {
				return fmt.Errorf("operation failed: %s", e.GetMessage())
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}
