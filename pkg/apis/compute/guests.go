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
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/billing"
)

type ServerFilterListInput struct {
	// 以关联主机（ID或Name）过滤列表
	Server string `json:"server"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	ServerId string `json:"server_id" deprecated-by:"server"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	Guest string `json:"guest" deprecated-by:"server"`
	// swagger:ignore
	// Deprecated
	// Filter by guest Id
	GuestId string `json:"guest_id" deprecated-by:"server"`
}

type ServerListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	HostFilterListInput

	NetworkFilterListInput

	billing.BillingResourceListInput

	GroupFilterListInput
	SecgroupFilterListInput
	DiskFilterListInput

	// 只列出裸金属主机
	Baremetal *bool `json:"baremetal"`
	// 只列出GPU主机
	Gpu *bool `json:"gpu"`
	// 列出管理安全组为指定安全组的主机
	AdminSecgroup string `json:"admin_security"`
	// 列出Hypervisor为指定值的主机
	// enum: kvm,esxi,baremetal,aliyun,azure,aws,huawei,ucloud,zstack,openstack,google,ctyun"`
	Hypervisor []string `json:"hypervisor"`
	// 列出绑定了弹性IP（EIP）的主机
	WithEip *bool `json:"with_eip"`
	// 列出未绑定弹性IP（EIO）的主机
	WithoutEip *bool `json:"without_eip"`
	// 列出操作系统为指定值的主机
	// enum: linux,windows,vmware
	OsType []string `json:"os_type"`

	// 对列表结果按照磁盘进行排序
	// enum: asc,desc
	// OrderByDisk string `json:"order_by_disk"`

	// 列出可以挂载指定EIP的主机
	UsableServerForEip string `json:"usable_server_for_eip"`

	// 按主机资源类型进行排序
	// enum: shared,prepaid,dedicated
	ResourceType string `json:"resource_type"`
	// 返回开启主备机功能的主机
	GetBackupGuestsOnHost *bool `json:"get_backup_guests_on_host"`

	// 根据宿主机 SN 过滤
	// HostSn string `json:"host_sn"`

	VcpuCount []int `json:"vcpu_count"`

	VmemSize []int `json:"vmem_size"`

	BootOrder []string `json:"boot_order"`

	Vga []string `json:"vga"`

	Vdi []string `json:"vdi"`

	Machine []string `json:"machine"`

	Bios []string `json:"bios"`

	SrcIpCheck *bool `json:"src_ip_check"`

	SrcMacCheck *bool `json:"src_mac_check"`

	InstanceType []string `json:"instance_type"`
}

func (input *ServerListInput) AfterUnmarshal() {
	if input.Baremetal != nil && *input.Baremetal {
		input.Hypervisor = append(input.Hypervisor, HYPERVISOR_BAREMETAL)
	}
}

type ServerRebuildRootInput struct {
	apis.Meta

	// 镜像名称
	Image string `json:"image"`
	// 镜像 id
	// required: true
	ImageId       string `json:"image_id"`
	Keypair       string `json:"keypair"`
	KeypairId     string `json:"keypair_id"`
	ResetPassword *bool  `json:"reset_password"`
	Password      string `json:"password"`
	AutoStart     *bool  `json:"auto_start"`
	AllDisks      *bool  `json:"all_disks"`
}

func (i ServerRebuildRootInput) GetImageName() string {
	if len(i.Image) > 0 {
		return i.Image
	}
	if len(i.ImageId) > 0 {
		return i.ImageId
	}
	return ""
}

func (i ServerRebuildRootInput) GetKeypairName() string {
	if len(i.Keypair) > 0 {
		return i.Keypair
	}
	if len(i.KeypairId) > 0 {
		return i.KeypairId
	}
	return ""
}

type ServerResumeInput struct {
	apis.Meta
}

type ServerDetails struct {
	apis.VirtualResourceDetails

	SGuest

	HostResourceInfo

	// details
	// 网络概要
	Networks string `json:"networks"`
	// 磁盘概要
	Disks string `json:"disks"`

	// 磁盘详情
	DisksInfo *jsonutils.JSONArray `json:"disks_info"`
	// 虚拟机Ip列表
	VirtualIps string `json:"virtual_ips"`
	// 安全组规则
	SecurityRules string `json:"security_rules"`
	// 操作系统名称
	OsName string `json:"os_name"`
	// 操作系统类型
	OsType string `json:"os_type"`
	// 系统管理员可见的安全组规则
	AdminSecurityRules string `json:"admin_security_rules"`

	// list
	AttachTime time.Time `attach_time`

	// common
	IsPrepaidRecycle bool `json:"is_prepaid_recycle"`

	// 备份主机所在宿主机名称
	BackupHostName string `json:"backup_host_name"`
	// 备份主机所在宿主机状态
	BackupHostStatus string `json:"backup_host_status"`

	// 是否可以回收
	CanRecycle bool `json:"can_recycle"`

	// 自动释放时间
	AutoDeleteAt time.Time `json:"auto_delete_at"`
	// 磁盘数量
	DiskCount int `json:"disk_count"`
	// 是否支持ISO启动
	CdromSupport bool `json:"cdrom_support"`

	// 磁盘大小
	// example:30720
	DiskSizeMb int64 `json:"disk"`
	// IP地址列表字符串
	// example: 10.165.2.1,172.16.8.1
	IPs string `json:"ips"`
	// 网卡信息
	Nics []GuestnetworkShortDesc `json:"nics"`

	// 归属VPC
	Vpc string `json:"vpc"`
	// 归属VPC ID
	VpcId string `json:"vpc_id"`

	// 关联安全组列表
	Secgroups []apis.StandaloneShortDesc `json:"secgroups"`
	// 关联主安全组
	Secgroup string `json:"secgroup"`

	// 浮动IP
	Eip string `json:"eip"`
	// 浮动IP类型
	EipMode string `json:"eip_mode"`

	// 密钥对
	Keypair string `json:"keypair"`

	// 直通设备（GPU）列表
	IsolatedDevices []SIsolatedDevice `json:"isolated_devices"`
	// 是否支持GPU
	IsGpu bool `json:"is_gpu"`

	// Cdrom信息
	Cdrom string `json:"cdrom,allowempty"`
}

type GuestJointResourceDetails struct {
	apis.VirtualJointResourceBaseDetails

	// 云主机名称
	Guest string `json:"guest"`
	// 云主机名称
	Server string `json:"server"`
}

type GuestJointsListInput struct {
	apis.VirtualJointResourceBaseListInput

	GuestFilterListInput
}

type GuestResourceInfo struct {
	// 虚拟机名称
	Guest string `json:"guest"`

	// 虚拟机状态
	GuestStatus string `json:"guest_status"`

	// 宿主机ID
	HostId string `json:"host_id"`

	HostResourceInfo
}

type GuestFilterListInput struct {
	HostFilterListInput

	// 以指定虚拟主机（ID或Name）过滤列表结果
	Server string `json:"server"`
	// swagger:ignore
	// Deprecated
	ServerId string `json:"server_id" deprecated-by:"server"`
	// swagger:ignore
	// Deprecated
	Guest string `json:"guest" deprecated-by:"server"`
	// swagger:ignore
	// Deprecated
	GuestId string `json:"guest_id" deprecated-by:"server"`

	// 以虚拟主机名称排序
	// pattern:asc|desc
	OrderByServer string `json:"order_by_server"`
}
