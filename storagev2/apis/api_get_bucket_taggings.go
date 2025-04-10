// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	getbuckettaggings "github.com/qiniu/go-sdk/v7/storagev2/apis/get_bucket_taggings"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type innerGetBucketTaggingsRequest getbuckettaggings.Request

func (query *innerGetBucketTaggingsRequest) getBucketName(ctx context.Context) (string, error) {
	return query.BucketName, nil
}
func (query *innerGetBucketTaggingsRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.BucketName != "" {
		allQuery.Set("bucket", query.BucketName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	return allQuery, nil
}

type GetBucketTaggingsRequest = getbuckettaggings.Request
type GetBucketTaggingsResponse = getbuckettaggings.Response

// 查询指定的存储空间已设置的标签信息
func (storage *Storage) GetBucketTaggings(ctx context.Context, request *GetBucketTaggingsRequest, options *Options) (*GetBucketTaggingsResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetBucketTaggingsRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	if innerRequest.Credentials == nil && storage.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	pathSegments := make([]string, 0, 1)
	pathSegments = append(pathSegments, "bucketTagging")
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
	uplogInterceptor, err := uplog.NewRequestUplog("getBucketTaggings", bucketName, "", func() (string, error) {
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
	var respBody GetBucketTaggingsResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
