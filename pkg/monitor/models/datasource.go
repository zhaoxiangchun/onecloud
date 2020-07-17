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

package models

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"golang.org/x/sync/errgroup"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/wait"
	"yunion.io/x/pkg/utils"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

var (
	DataSourceManager *SDataSourceManager
)

var (
	compile     = regexp.MustCompile(`\w{8}(-\w{4}){3}-\w{12}`)
	canShowTags = []string{"host", "zone", "region", "tenant", "brand"}
)

const (
	DefaultDataSource = "default"
)

const (
	ErrDataSourceDefaultNotFound = errors.Error("Default data source not found")
)

func init() {
	DataSourceManager = &SDataSourceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDataSource{},
			"datasources_tbl",
			"datasource",
			"datasources",
		),
	}
	DataSourceManager.SetVirtualObject(DataSourceManager)
	registry.RegisterService(DataSourceManager)
}

type SDataSourceManager struct {
	db.SStandaloneResourceBaseManager
}

func (_ *SDataSourceManager) IsDisabled() bool {
	return false
}

func (_ *SDataSourceManager) Init() error {
	return nil
}

func (man *SDataSourceManager) Run(ctx context.Context) error {
	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.Go(func() error { return man.initDefaultDataSource(ctx) })
	return errgrp.Wait()
}

func (man *SDataSourceManager) initDefaultDataSource(ctx context.Context) error {
	region := options.Options.Region
	initF := func() {
		ds, err := man.GetDefaultSource()
		if err != nil && err != ErrDataSourceDefaultNotFound {
			log.Errorf("Get default datasource: %v", err)
			return
		}
		if ds != nil {
			return
		}
		s := auth.GetAdminSessionWithPublic(ctx, region, "")
		if s == nil {
			log.Errorf("get empty public session for region %s", region)
			return
		}
		url, err := s.GetServiceURL("influxdb", identityapi.EndpointInterfacePublic)
		if err != nil {
			log.Errorf("get influxdb public url: %v", err)
			return
		}
		ds = &SDataSource{
			Type: monitor.DataSourceTypeInfluxdb,
			Url:  url,
		}
		ds.Name = DefaultDataSource
		if err := man.TableSpec().Insert(ctx, ds); err != nil {
			log.Errorf("insert default influxdb: %v", err)
		}
	}
	wait.Forever(initF, 30*time.Second)
	return nil
}

func (man *SDataSourceManager) GetDefaultSource() (*SDataSource, error) {
	obj, err := man.FetchByName(nil, DefaultDataSource)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDataSourceDefaultNotFound
		} else {
			return nil, err
		}
	}
	return obj.(*SDataSource), nil
}

type SDataSource struct {
	db.SStandaloneResourceBase

	Type      string            `nullable:"false" list:"user"`
	Url       string            `nullable:"false" list:"user"`
	User      string            `width:"64" charset:"utf8" nullable:"true"`
	Password  string            `width:"64" charset:"utf8" nullable:"true"`
	Database  string            `width:"64" charset:"utf8" nullable:"true"`
	IsDefault tristate.TriState `nullable:"false" default:"false" create:"optional"`
	/*
		TimeInterval string
		BasicAuth bool
		BasicAuthUser string
		BasicAuthPassword string
	*/
}

func (m *SDataSourceManager) GetSource(id string) (*SDataSource, error) {
	ret, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return ret.(*SDataSource), nil
}

func (ds *SDataSource) ToTSDBDataSource(db string) *tsdb.DataSource {
	if db == "" {
		db = ds.Database
	}
	return &tsdb.DataSource{
		Id:       ds.GetId(),
		Name:     ds.GetName(),
		Type:     ds.Type,
		Url:      ds.Url,
		User:     ds.User,
		Password: ds.Password,
		Database: db,
		Updated:  ds.UpdatedAt,
		/*BasicAuth: ds.BasicAuth,
		BasicAuthUser: ds.BasicAuthUser,
		BasicAuthPassword: ds.BasicAuthPassword,
		TimeInterval: ds.TimeInterval,*/
	}
}

func (self *SDataSourceManager) GetDatabases() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}
	db := influxdb.NewInfluxdb(dataSource.Url)
	//db.SetDatabase("telegraf")
	databases, err := db.GetDatabases()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "GetDatabases")
	}
	ret.Add(jsonutils.NewStringArray(databases), "databases")
	return ret, nil
}

func (self *SDataSourceManager) GetMeasurements(query jsonutils.JSONObject,
	measurementFilter, tagFilter string) (jsonutils.JSONObject,
	error) {
	ret := jsonutils.NewDict()
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, httperrors.NewInputParameterError("not support database")
	}
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}
	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(database)
	var buffer bytes.Buffer
	buffer.WriteString(" SHOW MEASUREMENTS ON ")
	buffer.WriteString(database)
	if len(measurementFilter) != 0 {
		buffer.WriteString(" WITH ")
		buffer.WriteString(measurementFilter)
	}
	if len(tagFilter) != 0 {
		buffer.WriteString(" WHERE ")
		buffer.WriteString(tagFilter)
	}
	dbRtn, err := db.Query(buffer.String())
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "SHOW MEASUREMENTS")
	}
	if len(dbRtn) > 0 && len(dbRtn[0]) > 0 {
		res := dbRtn[0][0]
		measurements := make([]monitor.InfluxMeasurement, len(res.Values))
		for i := range res.Values {
			tmpDict := jsonutils.NewDict()
			tmpDict.Add(res.Values[i][0], "measurement")
			err := tmpDict.Unmarshal(&measurements[i])
			measurements[i].Database = database
			if err != nil {
				return jsonutils.JSONNull, errors.Wrap(err, "measurement unmarshal error")
			}
		}
		startFilter := time.Now()
		filterMeasurements, err := self.filterMeasurementsByTime(db, measurements, query, true)
		if err != nil {
			return jsonutils.JSONNull, errors.Wrap(err, "filterMeasurementsByTime error")
		}
		log.Errorf("=====================filter end cost time is %f s", time.Now().Sub(startFilter).Seconds())
		ret.Add(jsonutils.Marshal(&filterMeasurements), "measurements")
	}
	return ret, nil
}

type influxdbQueryChan struct {
	queryRtnChan chan monitor.InfluxMeasurement
	count        int
}

func (self *SDataSourceManager) filterMeasurementsByTime(db *influxdb.SInfluxdb,
	measurements []monitor.InfluxMeasurement, query jsonutils.JSONObject, asynQury bool) ([]monitor.InfluxMeasurement,
	error) {
	timeF, err := self.getFromAndToFromParam(query)
	if err != nil {
		return nil, err
	}
	filterMeasurements := make([]monitor.InfluxMeasurement, 0)
	if asynQury {
		filterMeasurements, err = self.getFilterMeasurementsAsyn(timeF.From, timeF.To, measurements, db)
	} else {
		filterMeasurements, err = self.getfilterMeasurementsSyn(timeF.From, timeF.To, measurements, db)
	}
	if err != nil {
		return nil, err
	}
	return filterMeasurements, nil
}

type timeFilter struct {
	From string
	To   string
}

func (self *SDataSourceManager) getFromAndToFromParam(query jsonutils.JSONObject) (timeFilter, error) {
	timeF := timeFilter{}
	from, _ := query.GetString("form")
	if len(from) == 0 {
		from = "6h"
	}
	to, _ := query.GetString("to")
	if len(to) == 0 {
		to = "now"
	}
	timeFilter := monitor.AlertQuery{
		From: from,
		To:   to,
	}
	err := validators.ValidateFromAndToValue(timeFilter)
	if err != nil {
		return timeF, err
	}
	timeF.From = from
	timeF.To = to
	return timeF, nil
}

func (self *SDataSourceManager) getFilterMeasurementsAsyn(from, to string,
	measurements []monitor.InfluxMeasurement, db *influxdb.SInfluxdb) ([]monitor.InfluxMeasurement, error) {
	log.Errorln("start asynchronous task")
	filterMeasurements := make([]monitor.InfluxMeasurement, 0)
	queryChan := new(influxdbQueryChan)
	queryChan.queryRtnChan = make(chan monitor.InfluxMeasurement, len(measurements))
	queryChan.count = len(measurements)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	measurementQueryGroup, _ := errgroup.WithContext(ctx)
	for i, _ := range measurements {
		tmp := measurements[i]
		measurementQueryGroup.Go(func() error {
			return self.getFilterMeasurement(queryChan, from, to, tmp, db)
		})
	}
	measurementQueryGroup.Go(func() error {
		for i := 0; i < queryChan.count; i++ {
			select {
			case filterMeasurement := <-queryChan.queryRtnChan:
				if len(filterMeasurement.Measurement) != 0 {
					filterMeasurements = append(filterMeasurements, filterMeasurement)
				}
			case <-ctx.Done():
				return fmt.Errorf("filter measurement time out")
			}
		}
		return nil
	})
	err := measurementQueryGroup.Wait()
	return filterMeasurements, err
}

func (self *SDataSourceManager) getFilterMeasurement(queryChan *influxdbQueryChan, from, to string,
	measurement monitor.InfluxMeasurement, db *influxdb.SInfluxdb) error {
	rtnMeasurement := new(monitor.InfluxMeasurement)
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf(fmt.Sprintf("select count(*::field) from %s where %s ", measurement.Measurement,
		self.renderTimeFilter(from, to))))
	startQuery := time.Now()
	rtn, err := db.Query(buffer.String())
	log.Errorf("query cost time:%f s", time.Now().Sub(startQuery).Seconds())
	if err != nil {
		return errors.Wrap(err, "getFilterMeasurement error")
	}
	if len(rtn) != 0 && len(rtn[0]) != 0 {
		rtnMeasurement.Measurement = rtn[0][0].Name
	}
	queryChan.queryRtnChan <- *rtnMeasurement
	return nil
}

func (self *SDataSourceManager) getfilterMeasurementsSyn(from, to string,
	measurements []monitor.InfluxMeasurement, db *influxdb.SInfluxdb) ([]monitor.InfluxMeasurement, error) {
	var buffer bytes.Buffer
	for _, measurement := range measurements {
		buffer.WriteString(fmt.Sprintf("select *::field from %s where %s ", measurement.Measurement, self.renderTimeFilter(from, to)))
		buffer.WriteString(";")
	}
	if buffer.Len() == 0 {
		return measurements, nil
	}
	startQuery := time.Now()
	rtn, err := db.Query(buffer.String())
	log.Errorf("query cost time:%f s", time.Now().Sub(startQuery).Seconds())
	if err != nil {
		return nil, err
	}
	filterMeasurement := make([]monitor.InfluxMeasurement, 0)
	for _, result := range rtn {
		if len(result) != 0 {
			filterMeasurement = append(filterMeasurement, monitor.InfluxMeasurement{Measurement: result[0].Name})
		}
	}
	return filterMeasurement, nil
}

func (self *SDataSourceManager) renderTimeFilter(from, to string) string {
	if strings.Contains(from, "now-") {
		from = "now() - " + strings.Replace(from, "now-", "", 1)
	} else {
		from = "now() - " + from
	}

	tmp := ""
	if to != "now" && to != "" {
		tmp = " and time < now() - " + strings.Replace(to, "now-", "", 1)
	}

	return fmt.Sprintf("time > %s%s", from, tmp)

}

func (self *SDataSourceManager) GetMetricMeasurement(query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, httperrors.NewInputParameterError("not support database")
	}
	measurement, _ := query.GetString("measurement")
	if measurement == "" {
		return jsonutils.JSONNull, httperrors.NewInputParameterError("not support measurement")
	}
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}
	timeF, err := self.getFromAndToFromParam(query)
	if err != nil {
		return nil, err
	}
	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(database)
	output := new(monitor.InfluxMeasurement)
	output.Measurement = measurement
	output.Database = database
	output.TagValue = make(map[string][]string, 0)
	for _, val := range monitor.METRIC_ATTRI {
		err = getAttributesOnMeasurement(database, val, output, db)
		if err != nil {
			return jsonutils.JSONNull, errors.Wrap(err, "getAttributesOnMeasurement error")
		}
	}
	//err = getTagValue(database, output, db)
	tagValChan := influxdbTagValueChan{
		rtnChan: make(chan map[string][]string, len(output.FieldKey)),
		count:   len(output.FieldKey),
		//count: 1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	tagValGroup, _ := errgroup.WithContext(ctx)
	defer cancel()
	for i, _ := range output.FieldKey {

		tmpField := output.FieldKey[i]
		tmpMeasurement := *output
		tagValGroup.Go(func() error {
			return self.getFilterMeasurementTagValue(&tagValChan, timeF.From, timeF.To, tmpField, tmpMeasurement, db)
		})
	}
	tagValGroup.Go(func() error {
		for i := 0; i < tagValChan.count; i++ {
			select {
			case tagVal := <-tagValChan.rtnChan:
				if len(tagVal) != 0 {
					tagValUnion(output, tagVal)
				}
			case <-ctx.Done():
				return fmt.Errorf("filter getFilterMeasurementTagValue time out")
			}
		}
		return nil
	})
	err = tagValGroup.Wait()
	if err != nil {
		return jsonutils.JSONNull, err
	}
	return jsonutils.Marshal(output), nil

}

func tagValUnion(measurement *monitor.InfluxMeasurement, rtn map[string][]string) {
	for _, tag := range measurement.TagKey {
		if rtnTagVal, ok := rtn[tag]; ok {
			if _, ok := measurement.TagValue[tag]; !ok {
				measurement.TagValue[tag] = rtnTagVal
				continue
			}
			measurement.TagValue[tag] = union(measurement.TagValue[tag], rtnTagVal)
		}
	}
}

func union(slice1, slice2 []string) []string {
	m := make(map[string]int)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times, _ := m[v]
		if times == 0 {
			slice1 = append(slice1, v)
		}
	}
	return slice1
}

type InfluxdbSubscription struct {
	SubName  string
	DataBase string
	//retention policy
	Rc  string
	Url string
}

func (self *SDataSourceManager) AddSubscription(subscription InfluxdbSubscription) error {

	query := fmt.Sprintf("CREATE SUBSCRIPTION %s ON %s.%s DESTINATIONS ALL %s",
		jsonutils.NewString(subscription.SubName).String(),
		jsonutils.NewString(subscription.DataBase).String(),
		jsonutils.NewString(subscription.Rc).String(),
		strings.ReplaceAll(jsonutils.NewString(subscription.Url).String(), "\"", "'"),
	)
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return errors.Wrap(err, "s.GetDefaultSource")
	}

	db := influxdb.NewInfluxdbWithDebug(dataSource.Url, true)
	db.SetDatabase(subscription.DataBase)

	rtn, err := db.GetQuery(query)
	if err != nil {
		return err
	}
	for _, result := range rtn {
		for _, obj := range result {
			objJson := jsonutils.Marshal(&obj)
			log.Errorln(objJson.String())
		}
	}
	return nil
}

func (self *SDataSourceManager) DropSubscription(subscription InfluxdbSubscription) error {
	query := fmt.Sprintf("DROP SUBSCRIPTION %s ON %s.%s", jsonutils.NewString(subscription.SubName).String(),
		jsonutils.NewString(subscription.DataBase).String(),
		jsonutils.NewString(subscription.Rc).String(),
	)
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return errors.Wrap(err, "s.GetDefaultSource")
	}

	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(subscription.DataBase)
	rtn, err := db.Query(query)
	if err != nil {
		return err
	}
	for _, result := range rtn {
		for _, obj := range result {
			objJson := jsonutils.Marshal(&obj)
			log.Errorln(objJson.String())
		}
	}
	return nil
}

func getAttributesOnMeasurement(database, tp string, output *monitor.InfluxMeasurement, db *influxdb.SInfluxdb) error {
	dbRtn, err := db.Query(fmt.Sprintf("SHOW %s KEYS ON %s FROM %s", tp, database, output.Measurement))
	log.Errorf("SHOW %s KEYS ON %s FROM %s", tp, database, output.Measurement)
	if err != nil {
		return errors.Wrap(err, "SHOW MEASUREMENTS")
	}
	if len(dbRtn) == 0 || len(dbRtn[0]) == 0 {
		return nil
	}
	res := dbRtn[0][0]
	tmpDict := jsonutils.NewDict()
	tmpArr := jsonutils.NewArray()
	for i := range res.Values {
		v, _ := res.Values[i][0].(*jsonutils.JSONString).GetString()
		if filterTagKey(v) {
			continue
		}
		tmpArr.Add(res.Values[i][0])
	}
	tmpDict.Add(tmpArr, res.Columns[0])
	err = tmpDict.Unmarshal(output)
	if err != nil {
		return errors.Wrap(err, "measurement unmarshal error")
	}
	return nil
}

func getTagValue(database string, output *monitor.InfluxMeasurement, db *influxdb.SInfluxdb) error {
	if len(output.TagKey) == 0 {
		return nil
	}
	tagKeyStr := jsonutils.NewStringArray(output.TagKey).String()
	tagKeyStr = tagKeyStr[1 : len(tagKeyStr)-1]
	dbRtn, err := db.Query(fmt.Sprintf("SHOW TAG VALUES ON %s FROM %s WITH KEY IN (%s)", database, output.Measurement, tagKeyStr))
	if err != nil {
		return err
	}
	res := dbRtn[0][0]
	tagValue := make(map[string][]string, 0)
	keys := strings.Join(output.TagKey, ",")
	for i := range res.Values {
		val, _ := res.Values[i][0].(*jsonutils.JSONString).GetString()
		if !strings.Contains(keys, val) {
			continue
		}
		if _, ok := tagValue[val]; !ok {
			tagValue[val] = make([]string, 0)
		}
		tag, _ := res.Values[i][1].(*jsonutils.JSONString).GetString()
		if filterTagValue(tag) {
			delete(tagValue, val)
			continue
		}
		tagValue[val] = append(tagValue[val], tag)
	}
	output.TagValue = tagValue
	//TagKey == TagValue.keys
	tagK := make([]string, 0)
	for tag, _ := range output.TagValue {
		tagK = append(tagK, tag)
	}
	output.TagKey = tagK
	return nil
}

type influxdbTagValueChan struct {
	rtnChan chan map[string][]string
	count   int
}

func (self *SDataSourceManager) getFilterMeasurementTagValue(tagValueChan *influxdbTagValueChan, from string,
	to string, field string,
	measurement monitor.InfluxMeasurement, db *influxdb.SInfluxdb) error {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf(fmt.Sprintf(`SELECT mean("%s") FROM "%s" WHERE %s  GROUP BY * ,time(1m) fill(none)`,
		field, measurement.Measurement,
		self.renderTimeFilter(from, to))))
	log.Errorf("sql:", buffer.String())
	startQuery := time.Now()
	rtn, err := db.Query(buffer.String())
	log.Errorf("field:%s query cost time:%f s", field, time.Now().Sub(startQuery).Seconds())
	if err != nil {
		return errors.Wrap(err, "getFilterMeasurementTagValue query error")
	}
	tagValMap := make(map[string][]string)
	if len(rtn) != 0 && len(rtn[0]) != 0 {
		log.Errorf("start measurement.name:%s,measurement.name:%s", measurement.Measurement, rtn[0][0].Name)

		for rtnIndex, _ := range rtn {
			for serieIndex, _ := range rtn[rtnIndex] {
				tagMap, _ := rtn[rtnIndex][serieIndex].Tags.GetMap()
				for key, valObj := range tagMap {
					valStr, _ := valObj.GetString()
					if len(valStr) == 0 || valStr == "null" || filterTagValue(valStr) {
						continue
					}
					if !utils.IsInStringArray(key, measurement.TagKey) {
						//measurement.TagKey = append(measurement.TagKey, key)
						continue
					}
					if valArr, ok := tagValMap[key]; ok {
						if !utils.IsInStringArray(valStr, valArr) {
							tagValMap[key] = append(valArr, valStr)
						}
						continue
					}
					tagValMap[key] = []string{valStr}
				}
			}
		}
		measurement.TagValue = tagValMap
	}
	tagValueChan.rtnChan <- tagValMap
	return nil
}

var filterKey = []string{"perf_instance", "res_type", "status", "cloudregion", "os_type", "is_vm"}

func filterTagKey(key string) bool {
	if strings.Contains(key, "_id") {
		return true
	}
	if utils.IsInStringArray(key, filterKey) {
		return true
	}
	return false
}

func filterTagValue(val string) bool {
	if compile.MatchString(val) {
		return true
	}
	return false
}
