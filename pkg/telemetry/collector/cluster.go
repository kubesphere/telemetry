/*
Copyright 2023 The KubeSphere Authors.

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

package collector

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// collector cluster data

func init() {
	register(&Cluster{})
}

type Cluster struct {
	Role           string `json:"role"`
	Name           string `json:"name"`
	Uid            string `json:"uid"`
	Nid            string `json:"nid"`
	KSVersion      string `json:"ksVersion"`
	ClusterVersion string `json:"clusterVersion"`
	Namespace      int    `json:"namespace"`
	Nodes          []Node `json:"nodes"`
}

type Node struct {
	Uid              string   `json:"uid"`
	Name             string   `json:"name"`
	Role             []string `json:"role"`
	Arch             string   `json:"arch"`
	ContainerRuntime string   `json:"containerRuntime"`
	Kernel           string   `json:"kernel"`
	KubeProxy        string   `json:"kubeProxy"`
	Kubelet          string   `json:"kubelet"`
	Os               string   `json:"os"`
	OsImage          string   `json:"osImage"`
}

func (c Cluster) RecordKey() string {
	return "clusters"
}

func (c Cluster) Collect(ctx context.Context, client runtimeClient.Client) (interface{}, error) {
	var clusterList = &clusterv1alpha1.ClusterList{}
	if err := client.List(ctx, clusterList); err != nil {
		return c, nil
	}
	// statistics cluster Data
	resCluster := make([]Cluster, len(clusterList.Items))
	for i, cluster := range clusterList.Items {
		if string(cluster.Status.UID) == "" {
			return nil, fmt.Errorf("collector cluster  %s error. cluster is not ready", cluster.Name)
		}
		resCluster[i] = Cluster{
			Name:           cluster.Name,
			Uid:            string(cluster.UID),
			Nid:            string(cluster.Status.UID),
			KSVersion:      cluster.Status.KubeSphereVersion,
			ClusterVersion: cluster.Status.KubernetesVersion,
		}
		if _, ok := cluster.Labels[clusterv1alpha1.HostCluster]; ok {
			resCluster[i].Role = "host"
		} else {
			resCluster[i].Role = "member"
		}
		kubeClient, err := c.getKubeClient(cluster.Spec.Connection.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("get kube client from cluster %v error %v", cluster.Name, err)
		}
		resCluster[i].Namespace = c.getNamespace(ctx, kubeClient)
		resCluster[i].Nodes = c.getNodes(ctx, kubeClient)
	}
	return resCluster, nil
}

func (c Cluster) getKubeClient(config []byte) (kubernetes.Interface, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes(config)
	if err != nil {
		return nil, err
	}
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		klog.Errorf("get cluster rest config error %v", err)
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Errorf("get cluster kube config error %v", err)
		return nil, err
	}
	return kubeClient, nil
}
func (c Cluster) getNamespace(ctx context.Context, kubeClient kubernetes.Interface) int {
	namespaceList, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{TimeoutSeconds: pointer.Int64(30)})
	if err != nil {
		klog.Errorf("list namespace error %v", err)
		return 0
	}
	return len(namespaceList.Items)
}

func (c Cluster) getNodes(ctx context.Context, kubeClient kubernetes.Interface) []Node {
	nodeList, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{TimeoutSeconds: pointer.Int64(30)})
	if err != nil {
		klog.Errorf("get node list from cluster kube config error %v", err)
		return nil
	}
	// statistics node data
	resNode := make([]Node, len(nodeList.Items))
	for i, node := range nodeList.Items {
		roles := make([]string, 0)
		for k := range node.Labels {
			if strings.HasPrefix(k, "node-role.kubernetes.io/") {
				roles = append(roles, k[24:])
			}
		}
		resNode[i] = Node{
			Uid:              string(node.UID),
			Name:             node.Name,
			Role:             roles,
			Arch:             node.Status.NodeInfo.Architecture,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			Kernel:           node.Status.NodeInfo.KernelVersion,
			KubeProxy:        node.Status.NodeInfo.KubeProxyVersion,
			Kubelet:          node.Status.NodeInfo.KubeletVersion,
			Os:               node.Status.NodeInfo.OperatingSystem,
			OsImage:          node.Status.NodeInfo.OSImage,
		}
	}
	return resNode
}
