// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	getregions "github.com/qiniu/go-sdk/v7/storagev2/apis/get_regions"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/http"
	"strings"
	"time"
)

type innerGetRegionsRequest getregions.Request
type GetRegionsRequest = getregions.Request
type GetRegionsResponse = getregions.Response

// 获取所有区域信息
func (storage *Storage) GetRegions(ctx context.Context, request *GetRegionsRequest, options *Options) (*GetRegionsResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetRegionsRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	if innerRequest.Credentials == nil && storage.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	pathSegments := make([]string, 0, 1)
	pathSegments = append(pathSegments, "regions")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	headers := http.Header{}
	bucketName := options.OverwrittenBucketName
	uplogInterceptor, err := uplog.NewRequestUplog("getRegions", bucketName, "", func() (string, error) {
		credentials := innerRequest.Credentials
		if credentials == nil {
			credentials = storage.client.GetCredentials()
		}
		putPolicy, err := uptoken.NewPutPolicy(bucketName, time.Now().Add(time.Hour))
		if err != nil {
			return "", err
		}
		return uptoken.NewSigner(putPolicy, credentials).GetUpToken(ctx)
	})
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, Header: headers, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true, OnRequestProgress: options.OnRequestProgress}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		bucketHosts := httpclient.DefaultBucketHosts()
		if options.OverwrittenBucketHosts != nil {
			req.Endpoints = options.OverwrittenBucketHosts
		} else {
			req.Endpoints = bucketHosts
		}
	}
	var respBody GetRegionsResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
