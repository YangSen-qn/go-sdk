// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	addbucketeventrule "github.com/qiniu/go-sdk/v7/storagev2/apis/add_bucket_event_rule"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type innerAddBucketEventRuleRequest addbucketeventrule.Request

func (query *innerAddBucketEventRuleRequest) getBucketName(ctx context.Context) (string, error) {
	return query.Bucket, nil
}
func (query *innerAddBucketEventRuleRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.Bucket != "" {
		allQuery.Set("bucket", query.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if query.Name != "" {
		allQuery.Set("name", query.Name)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Name"}
	}
	if query.Prefix != "" {
		allQuery.Set("prefix", query.Prefix)
	}
	if query.Suffix != "" {
		allQuery.Set("suffix", query.Suffix)
	}
	if len(query.EventTypes) > 0 {
		for _, value := range query.EventTypes {
			allQuery.Add("event", value)
		}
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "EventTypes"}
	}
	if len(query.CallbackUrls) > 0 {
		for _, value := range query.CallbackUrls {
			allQuery.Add("callbackURL", value)
		}
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "CallbackUrls"}
	}
	if query.AccessKey != "" {
		allQuery.Set("accessKey", query.AccessKey)
	}
	if query.Host != "" {
		allQuery.Set("host", query.Host)
	}
	return allQuery, nil
}

type AddBucketEventRuleRequest = addbucketeventrule.Request
type AddBucketEventRuleResponse = addbucketeventrule.Response

// 增加存储空间事件通知规则
func (storage *Storage) AddBucketEventRule(ctx context.Context, request *AddBucketEventRuleRequest, options *Options) (*AddBucketEventRuleResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerAddBucketEventRuleRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	if innerRequest.Credentials == nil && storage.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	pathSegments := make([]string, 0, 2)
	pathSegments = append(pathSegments, "events", "add")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	if query, err := innerRequest.buildQuery(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	headers := http.Header{}
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	uplogInterceptor, err := uplog.NewRequestUplog("addBucketEventRule", bucketName, "", func() (string, error) {
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
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, Header: headers, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, OnRequestProgress: options.OnRequestProgress}
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
	return &AddBucketEventRuleResponse{}, resp.Body.Close()
}
