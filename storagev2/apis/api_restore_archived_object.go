// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	restorearchivedobject "github.com/qiniu/go-sdk/v7/storagev2/apis/restore_archived_object"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type innerRestoreArchivedObjectRequest restorearchivedobject.Request

func (pp *innerRestoreArchivedObjectRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.Entry, ":", 2)[0], nil
}
func (pp *innerRestoreArchivedObjectRequest) getObjectName() string {
	parts := strings.SplitN(pp.Entry, ":", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
func (path *innerRestoreArchivedObjectRequest) buildPath() ([]string, error) {
	allSegments := make([]string, 0, 3)
	if path.Entry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.Entry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Entry"}
	}
	if path.FreezeAfterDays != 0 {
		allSegments = append(allSegments, "freezeAfterDays", strconv.FormatInt(path.FreezeAfterDays, 10))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "FreezeAfterDays"}
	}
	return allSegments, nil
}
func (request *innerRestoreArchivedObjectRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type RestoreArchivedObjectRequest = restorearchivedobject.Request
type RestoreArchivedObjectResponse = restorearchivedobject.Response

// 解冻归档存储类型的文件，可设置解冻有效期1～7天，完成解冻任务通常需要1～5分钟
func (storage *Storage) RestoreArchivedObject(ctx context.Context, request *RestoreArchivedObjectRequest, options *Options) (*RestoreArchivedObjectResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerRestoreArchivedObjectRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRs}
	if innerRequest.Credentials == nil && storage.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	pathSegments := make([]string, 0, 4)
	pathSegments = append(pathSegments, "restoreAr")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	headers := http.Header{}
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	objectName := innerRequest.getObjectName()
	uplogInterceptor, err := uplog.NewRequestUplog("restoreArchivedObject", bucketName, objectName, func() (string, error) {
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
		if bucketName != "" {
			query := storage.client.GetBucketQuery()
			if query == nil {
				if options.OverwrittenBucketHosts != nil {
					if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
						return nil, err
					}
				}
				queryOptions := region.BucketRegionsQueryOptions{UseInsecureProtocol: storage.client.UseInsecureProtocol(), AccelerateUploading: storage.client.AccelerateUploadingEnabled(), HostFreezeDuration: storage.client.GetHostFreezeDuration(), Resolver: storage.client.GetResolver(), Chooser: storage.client.GetChooser(), BeforeResolve: storage.client.GetBeforeResolveCallback(), AfterResolve: storage.client.GetAfterResolveCallback(), ResolveError: storage.client.GetResolveErrorCallback(), BeforeBackoff: storage.client.GetBeforeBackoffCallback(), AfterBackoff: storage.client.GetAfterBackoffCallback(), BeforeRequest: storage.client.GetBeforeRequestCallback(), AfterResponse: storage.client.GetAfterResponseCallback()}
				if hostRetryConfig := storage.client.GetHostRetryConfig(); hostRetryConfig != nil {
					queryOptions.RetryMax = hostRetryConfig.RetryMax
					queryOptions.Backoff = hostRetryConfig.Backoff
				}
				if query, err = region.NewBucketRegionsQuery(bucketHosts, &queryOptions); err != nil {
					return nil, err
				}
			}
			if query != nil {
				var accessKey string
				var err error
				if accessKey, err = innerRequest.getAccessKey(ctx); err != nil {
					return nil, err
				}
				if accessKey == "" {
					if credentialsProvider := storage.client.GetCredentials(); credentialsProvider != nil {
						if creds, err := credentialsProvider.Get(ctx); err != nil {
							return nil, err
						} else if creds != nil {
							accessKey = creds.AccessKey
						}
					}
				}
				if accessKey != "" {
					req.Region = query.Query(accessKey, bucketName)
				}
			}
		}
	}
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &RestoreArchivedObjectResponse{}, resp.Body.Close()
}
