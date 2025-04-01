// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	setbucketcorsrules "github.com/qiniu/go-sdk/v7/storagev2/apis/set_bucket_cors_rules"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/http"
	"strings"
	"time"
)

type innerSetBucketCORSRulesRequest setbucketcorsrules.Request

func (pp *innerSetBucketCORSRulesRequest) getBucketName(ctx context.Context) (string, error) {
	return pp.Bucket, nil
}
func (path *innerSetBucketCORSRulesRequest) buildPath() ([]string, error) {
	allSegments := make([]string, 0, 1)
	if path.Bucket != "" {
		allSegments = append(allSegments, path.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	return allSegments, nil
}
func (j *innerSetBucketCORSRulesRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*setbucketcorsrules.Request)(j))
}
func (j *innerSetBucketCORSRulesRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*setbucketcorsrules.Request)(j))
}

type SetBucketCORSRulesRequest = setbucketcorsrules.Request
type SetBucketCORSRulesResponse = setbucketcorsrules.Response

// 设置空间的跨域规则
func (storage *Storage) SetBucketCORSRules(ctx context.Context, request *SetBucketCORSRulesRequest, options *Options) (*SetBucketCORSRulesResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerSetBucketCORSRulesRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	if innerRequest.Credentials == nil && storage.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	pathSegments := make([]string, 0, 3)
	pathSegments = append(pathSegments, "corsRules", "set")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	headers := http.Header{}
	body, err := httpclient.GetJsonRequestBody(&innerRequest)
	if err != nil {
		return nil, err
	}
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	uplogInterceptor, err := uplog.NewRequestUplog("setBucketCorsrules", bucketName, "", func() (string, error) {
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
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, Header: headers, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, RequestBody: body, OnRequestProgress: options.OnRequestProgress}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		bucketHosts := httpclient.DefaultBucketHosts()
		if options.OverwrittenBucketHosts != nil {
			req.Endpoints = options.OverwrittenBucketHosts
		} else {
			req.Endpoints = bucketHosts
		}
	}
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &SetBucketCORSRulesResponse{}, resp.Body.Close()
}
