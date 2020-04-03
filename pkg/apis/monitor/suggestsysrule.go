package monitor

import (
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	EIP_UN_USED = "EIP_UNUSED"

	DRIVER_ACTION = "DELETE"
)

type SuggestSysRuleListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput
}

type SuggestSysRuleCreateInput struct {
	apis.VirtualResourceCreateInput

	// 查询指标周期
	Period  string                   `json:"period"`
	Type    string                   `json:"type"`
	Enabled *bool                    `json:"enabled"`
	Setting *SSuggestSysAlertSetting `json:"setting"`
}

type SuggestSysRuleUpdateInput struct {
	apis.Meta

	// 查询指标周期
	Period   string                   `json:"period"`
	Type     string                   `json:"type"`
	Setting  *SSuggestSysAlertSetting `json:"setting"`
	Enabled  *bool                    `json:"enabled"`
	ExecTime time.Time                `json:"exec_time"`
}

type SuggestSysRuleDetails struct {
	apis.VirtualResourceDetails

	ID      string                   `json:"id"`
	Name    string                   `json:"name"`
	Setting *SSuggestSysAlertSetting `json:"setting"`
	Enabled bool                     `json:"enabled"`
}

type SSuggestSysAlertSetting struct {
	EIPUnused *EIPUnused `json:"eip_unused"`
}

type EIPUnused struct {
	Status string `json:"status"`
}

func (rule *EIPUnused) Validate() error {
	if len(rule.Status) == 0 {
		return errors.Wrap(httperrors.ErrEmptyRequest, "status")
	}
	return nil
}
