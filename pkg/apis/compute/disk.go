// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type DiskCreateInput struct {
	apis.VirtualResourceCreateInput

	*DiskConfig

	// 此参数仅适用于未指定storage时进行调度到指定区域创建磁盘
	// required: false
	PreferRegion string `json:"prefer_region_id"`

	// 此参数仅适用于未指定storage时进行调度到指定可用区区创建磁盘
	// required: false
	PreferZone string `json:"prefer_zone_id"`

	// swagger:ignore
	PreferWire string `json:"prefer_wire_id"`

	// 此参数仅适用于未指定storage时进行调度到指定可用区区创建磁盘
	// required: false
	PreferHost string `json:"prefer_host_id"`

	// 此参数仅适用于未指定storage时进行调度到指定平台创建磁盘
	// default: kvm
	// enum: kvm, openstack, esxi, aliyun, aws, qcloud, azure, huawei, openstack, ucloud, zstack google, ctyun
	Hypervisor string `json:"hypervisor"`
}

// ToServerCreateInput used by disk schedule
func (req *DiskCreateInput) ToServerCreateInput() *ServerCreateInput {
	input := ServerCreateInput{
		ServerConfigs: &ServerConfigs{
			PreferRegion: req.PreferRegion,
			PreferZone:   req.PreferZone,
			PreferWire:   req.PreferWire,
			PreferHost:   req.PreferHost,
			Hypervisor:   req.Hypervisor,
			Disks:        []*DiskConfig{req.DiskConfig},
			// Project:      req.Project,
			// Domain:       req.Domain,
		},
	}
	input.Name = req.Name
	input.Project = req.Project
	input.ProjectId = req.ProjectId
	input.Domain = req.Domain
	input.DomainId = req.DomainId
	return &input
}

func (req *ServerCreateInput) ToDiskCreateInput() *DiskCreateInput {
	input := DiskCreateInput{
		DiskConfig:   req.Disks[0],
		PreferRegion: req.PreferRegion,
		PreferHost:   req.PreferHost,
		PreferZone:   req.PreferZone,
		PreferWire:   req.PreferWire,
		Hypervisor:   req.Hypervisor,
	}
	input.Name = req.Name
	input.Project = req.Project
	input.Domain = req.Domain
	return &input
}

type SnapshotPolicyFilterListInput struct {
	// filter disk by snapshotpolicy
	Snapshotpolicy string `json:"snapshotpolicy"`
	// swagger:ignore
	// Deprecated
	// filter disk by snapshotpolicy_id
	SnapshotpolicyId string `json:"snapshotpolicy_id" deprecated-by:"snapshotpolicy"`
}

type DiskListInput struct {
	apis.VirtualResourceListInput

	ManagedResourceListInput

	BillingFilterListInput
	StorageFilterListInput
	StorageShareFilterListInput
	SnapshotPolicyFilterListInput
	ServerFilterListInput

	// filter disk by whether it is being used
	Unused *bool `json:"unused"`

	// swagger:ignore
	// Deprecated
	// filter by disk type
	Type string `json:"type" deprecated-by:"disk_type"`
	// 过滤指定disk_type的磁盘列表，可能的值为：sys, data, swap. volume
	//
	// | disk_type值 | 说明 |
	// |-------------|----------|
	// | sys         | 虚拟机系统盘    |
	// | data        | 虚拟机数据盘    |
	// | swap        | 虚拟机内存交换盘 |
	// | volume      | 容器volumn盘   |
	//
	DiskType string `json:"disk_type"`
}

type DiskFilterListInput struct {
	// 以指定虚拟磁盘（ID或Name）过滤列表结果
	Disk string `json:"disk"`
	// swagger:ignore
	// Deprecated
	// filter by disk_id
	DiskId string `json:"disk_id" deprecated-by:"disk"`
}
