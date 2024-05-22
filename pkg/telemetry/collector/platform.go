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

	iamv1beta1 "kubesphere.io/api/iam/v1beta1"
	tenantv1beta1 "kubesphere.io/api/tenant/v1beta1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	register(&Project{})
}

type Project struct {
	Workspace int `json:"workspace"`
	User      int `json:"user"`
}

func (p Project) RecordKey() string {
	return "platform"
}

func (p Project) Collect(ctx context.Context, client runtimeClient.Client) (interface{}, error) {
	workspaceList := &tenantv1beta1.WorkspaceList{}
	userList := &iamv1beta1.UserList{}

	// counting the number of workspace
	if err := client.List(ctx, workspaceList); err != nil {
		return nil, err
	}
	p.Workspace = len(workspaceList.Items)

	// counting the number of user
	if err := client.List(context.Background(), userList); err != nil {
		return nil, err
	}
	p.User = len(userList.Items)

	return p, nil
}
