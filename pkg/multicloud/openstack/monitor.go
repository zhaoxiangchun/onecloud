package openstack

import (
	net_url "net/url"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/timeutils"
)

func (region *SRegion) GetMonitorData(name, instanceId string, since time.Time,
	until time.Time) (jsonutils.JSONObject, error) {
	url := "/v1/resource/generic/" + instanceId + "/metric/" + name + "/measures"
	values := net_url.Values{}
	values.Add("start", since.Format(timeutils.IsoTimeFormat))
	values.Add("end", until.Format(timeutils.IsoTimeFormat))
	url += "?" + values.Encode()
	_, resp, err := region.Get("metric", url, "", nil)
	return resp, err
}
