package monitor

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type SuggestSysAlertListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	//监控规则type：Rule Type
	Type  string `json:"type"`
	ResId string `json:"res_id"`
}

type SuggestSysAlertCreateInput struct {
	apis.VirtualResourceCreateInput

	Enabled       *bool                    `json:"enabled"`
	MonitorConfig *SSuggestSysAlertSetting `json:"monitor_config"`

	//转换成ResId
	ResID string `json:"res_id"`
	Type  string `json:"type"`
	//Problem jsonutils.JSONObject `json:"problem"`
	Suggest string `json:"suggest"`
	Action  string `json:"action"`

	RuleAt time.Time `json:"rule_at"`
}

type SuggestSysAlertDetails struct {
	apis.VirtualResourceDetails
}

type SuggestSysAlertUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	Enabled       *bool                    `json:"enabled"`
	MonitorConfig *SSuggestSysAlertSetting `json:"monitor_config"`

	//转换成ResId
	ResID string `json:"res_id"`
	Type  string `json:"type"`
	//Problem jsonutils.JSONObject `json:"problem"`
	Suggest string `json:"suggest"`
	Action  string `json:"action"`

	RuleAt time.Time `json:"rule_at"`
}
