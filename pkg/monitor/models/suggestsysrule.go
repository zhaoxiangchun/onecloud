package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	SuggestSysRuleManager *SSuggestSysRuleManager
	//存储初始化的内容，同时起到默认配置的作用。
	suggestSysRuleDrivers = make(map[string]ISuggestSysRuleDriver, 0)
)

func init() {
	SuggestSysRuleManager = &SSuggestSysRuleManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&DSuggestSysAlert{},
			"suggestsysrule_tbl",
			"suggestsysrule",
			"suggestsysrules",
		),
	}
	SuggestSysRuleManager.SetVirtualObject(SuggestSysRuleManager)
}

type ISuggestSysRuleDriver interface {
	GetType() string
	Run(instance *monitor.SSuggestSysAlertSetting)
	ValidateSetting(input *monitor.SSuggestSysAlertSetting) error
	DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool)
}

func RegisterSuggestSysRuleDrivers(drvs ...ISuggestSysRuleDriver) {
	for _, drv := range drvs {
		suggestSysRuleDrivers[drv.GetType()] = drv
	}
}

func GetSuggestSysRuleDrivers() map[string]ISuggestSysRuleDriver {
	return suggestSysRuleDrivers
}

type SSuggestSysRuleManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type DSuggestSysRuleConfig struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	Type     string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Period   string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Setting  jsonutils.JSONObject ` list:"user" update:"user"`
	ExecTime time.Time            `json:"exec_time"`
}

func (man *SSuggestSysRuleManager) FetchSuggestSysAlartSettings(ruleTypes ...string) (map[string]*monitor.SSuggestSysAlertSetting, error) {
	objs := make([]*DSuggestSysRuleConfig, 0)
	suggestSysAlerSettingMap := make(map[string]*monitor.SSuggestSysAlertSetting, 0)
	q := man.Query()
	if q == nil {
		fmt.Println(" query is nil")
	}
	if len(ruleTypes) != 0 {
		q.Equals("type", ruleTypes)
	}
	err := db.FetchModelObjects(man, q, &objs)
	if err != nil && err != sql.ErrNoRows {
		return suggestSysAlerSettingMap, errors.Wrap(err, "FetchSuggestSysAlartSettings")
	}
	for _, config := range objs {
		setting, err := config.getSuggestSysAlertSetting()
		if err != nil {
			return suggestSysAlerSettingMap, errors.Wrap(err, "FetchSuggestSysAlartSettings")
		}
		suggestSysAlerSettingMap[config.Type] = setting
	}
	return suggestSysAlerSettingMap, nil
}

//根据数据库中查询得到的信息进行适配转换，同时更新drivers中的内容
func (dConfig *DSuggestSysRuleConfig) getSuggestSysAlertSetting() (*monitor.SSuggestSysAlertSetting, error) {
	setting := new(monitor.SSuggestSysAlertSetting)
	switch dConfig.Type {
	case monitor.EIP_UN_USED:
		setting.EIPUnused = new(monitor.EIPUnused)
		err := dConfig.Setting.Unmarshal(setting.EIPUnused)
		if err != nil {
			return nil, errors.Wrap(err, "DSuggestSysRuleConfig getSuggestSysAlertSetting error")
		}
	}
	return setting, nil
}

type DiskUnsed struct {
	Status string
}

func (manager *SSuggestSysRuleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.SuggestSysRuleListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (man *SSuggestSysRuleManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.SuggestSysRuleCreateInput) (*monitor.SuggestSysRuleCreateInput, error) {
	if data.Period == "" {
		// default 30s
		data.Period = "30s"
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	if dri, ok := suggestSysRuleDrivers[data.Type]; !ok {
		return nil, httperrors.NewInputParameterError("not support type %q", data.Type)
	} else {
		err := dri.ValidateSetting(data.Setting)
		if err != nil {
			return nil, errors.Wrap(err, "validate setting error")
		}
	}
	return &data, nil
}

func (rule *DSuggestSysRuleConfig) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data monitor.SuggestSysRuleUpdateInput) (monitor.SuggestSysRuleUpdateInput, error) {
	if data.Period == "" {
		// default 30s
		data.Period = "30s"
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return data, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	err := suggestSysRuleDrivers[rule.Type].ValidateSetting(data.Setting)
	if err != nil {
		return data, errors.Wrap(err, "validate setting error")
	}
	return data, nil
}

func (man *SSuggestSysRuleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.SuggestSysRuleDetails {
	rows := make([]monitor.SuggestSysRuleDetails, len(objs))
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.SuggestSysRuleDetails{
			VirtualResourceDetails: virtRows[i],
		}
		rows[i] = objs[i].(*DSuggestSysRuleConfig).getMoreDetails(rows[i])
	}
	return rows
}

func (self *DSuggestSysRuleConfig) getMoreDetails(out monitor.SuggestSysRuleDetails) monitor.SuggestSysRuleDetails {
	var err error
	out.Setting, err = self.getSuggestSysAlertSetting()
	if err != nil {
		log.Errorln("getMoreDetails err:", err)
	}
	out.ID = self.Id
	out.Name = self.Name
	out.Enabled = self.GetEnabled()
	return out
}

//after create, update Cronjob's info
func (self *DSuggestSysRuleConfig) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	cronman.GetCronJobManager().Remove(self.Name)
	if self.Enabled.Bool() {
		dur, _ := time.ParseDuration(self.Period)
		cronman.GetCronJobManager().AddJobAtIntervalsWithStartRun(self.Name, dur,
			suggestSysRuleDrivers[self.Type].DoSuggestSysRule, true)
	}
}

//after update, update Cronjob's info
func (self *DSuggestSysRuleConfig) PostUpdate(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	cronman.GetCronJobManager().Remove(self.Name)
	if self.Enabled.Bool() {
		dur, _ := time.ParseDuration(self.Period)
		cronman.GetCronJobManager().AddJobAtIntervalsWithStartRun(self.Name, dur,
			suggestSysRuleDrivers[self.Name].DoSuggestSysRule, true)
	}
}
