package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/middleware"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/util"
)

func NewReverseProxy(ds *m.DataSource, proxyPath string) *httputil.ReverseProxy {
	target, _ := url.Parse(ds.Url)

	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		reqQueryVals := req.URL.Query()

		if ds.Type == m.DS_INFLUXDB_08 {
			req.URL.Path = util.JoinUrlFragments(target.Path, "db/"+ds.Database+"/"+proxyPath)
			reqQueryVals.Add("u", ds.User)
			reqQueryVals.Add("p", ds.Password)
			req.URL.RawQuery = reqQueryVals.Encode()
		} else if ds.Type == m.DS_INFLUXDB {
			req.URL.Path = util.JoinUrlFragments(target.Path, proxyPath)
			reqQueryVals.Add("db", ds.Database)
			reqQueryVals.Add("u", ds.User)
			reqQueryVals.Add("p", ds.Password)
			req.URL.RawQuery = reqQueryVals.Encode()
		} else {
			req.URL.Path = util.JoinUrlFragments(target.Path, proxyPath)
		}

		if ds.BasicAuth {
			req.Header.Add("Authorization", util.GetBasicAuthHeader(ds.BasicAuthUser, ds.BasicAuthPassword))
		}
	}

	return &httputil.ReverseProxy{Director: director}
}

//ProxyDataSourceRequest TODO need to cache datasources
func ProxyDataSourceRequest(c *middleware.Context) {
	id := c.ParamsInt64(":id")
	query := m.GetDataSourceByIdQuery{Id: id, OrgId: c.OrgId}

	if err := bus.Dispatch(&query); err != nil {
		c.JsonApiErr(500, "Unable to load datasource meta data", err)
		return
	}

	proxyPath := c.Params("*")
	proxy := NewReverseProxy(&query.Result, proxyPath)
	proxy.ServeHTTP(c.RW(), c.Req.Request)
}
