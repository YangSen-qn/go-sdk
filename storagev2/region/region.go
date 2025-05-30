package region

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/freezer"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
)

type (
	// 服务地址
	//
	// 可以存储域名或 IP，端口和协议可选
	Endpoints struct {
		Preferred   []string `json:"preferred,omitempty"`   // 首选服务地址
		Alternative []string `json:"alternative,omitempty"` // 备选服务地址
		Accelerated []string `json:"accelerated,omitempty"` // 加速服务地址
	}

	// 区域信息
	//
	// 可能有多个机房信息，每个机房可能有多个服务地址
	//
	// 如果使用公有云，建议使用 GetRegionByID 方法直接获取获取区域实例，不建议手动设置服务地址
	Region struct {
		RegionID string    `json:"region_id,omitempty"` // 区域 ID
		Up       Endpoints `json:"up,omitempty"`        // Up 服务域名
		Io       Endpoints `json:"io,omitempty"`        // Io 服务域名
		IoSrc    Endpoints `json:"io_src,omitempty"`    // Io 源站服务域名
		Rs       Endpoints `json:"rs,omitempty"`        // Rs 服务域名
		Rsf      Endpoints `json:"rsf,omitempty"`       // Rsf 服务域名
		Api      Endpoints `json:"api,omitempty"`       // Api 服务域名
		Bucket   Endpoints `json:"bucket,omitempty"`    // Bucket 服务域名
	}

	// 区域提供者
	RegionsProvider interface {
		GetRegions(context.Context) ([]*Region, error)
	}

	// 服务名称
	ServiceName string

	endpointsStatus uint8

	// 服务地址迭代器
	EndpointsIter struct {
		endpoints Endpoints
		index     int
		current   endpointsStatus
	}

	// 服务地址提供者
	EndpointsProvider interface {
		GetEndpoints(context.Context) (Endpoints, error)
	}

	endpointsHostProvider struct {
		iter          *EndpointsIter
		freezer       freezer.Freezer
		lastFreezeErr error
	}
)

const (
	endpointsStatusAccelerated endpointsStatus = iota
	endpointsStatusPreferred
	endpointsStatusAlternative
)

const (
	// Up 服务
	ServiceUp ServiceName = "up"
	// Io 服务
	ServiceIo ServiceName = "io"
	// Io 源站服务
	ServiceIoSrc ServiceName = "io_src"
	// Rs 服务
	ServiceRs ServiceName = "rs"
	// Rsf 服务
	ServiceRsf ServiceName = "rsf"
	// Api 服务
	ServiceApi ServiceName = "api"
	// Bucket 服务
	ServiceBucket ServiceName = "bucket"
)

var (
	ErrUnrecognizedServiceName = errors.New("unrecognized service name")
)

// 根据 RegionID 获取公有云区域信息
func GetRegionByID(regionID string, useHttps bool) *Region {
	region := &Region{RegionID: regionID}
	if regionID == "z0" {
		region.Up.Preferred = []string{makeHost("upload.qiniup.com", useHttps), makeHost("upload-z0.qiniup.com", useHttps), makeHost("up.qiniup.com", useHttps), makeHost("up-z0.qiniup.com", useHttps)}
		region.Up.Alternative = []string{makeHost("up.qbox.me", useHttps), makeHost("up-z0.qbox.me", useHttps)}
		region.Io.Preferred = []string{makeHost("iovip.qiniuio.com", useHttps), makeHost("iovip-z0.qiniuio.com", useHttps)}
		region.Io.Alternative = []string{makeHost("iovip.qbox.me", useHttps), makeHost("iovip-z0.qbox.me", useHttps)}
	} else {
		region.Up.Preferred = []string{makeHost(fmt.Sprintf("upload-%s.qiniup.com", regionID), useHttps), makeHost(fmt.Sprintf("up-%s.qiniup.com", regionID), useHttps)}
		region.Io.Preferred = []string{makeHost(fmt.Sprintf("iovip-%s.qiniuio.com", regionID), useHttps)}
	}
	region.Rs.Preferred = []string{makeHost(fmt.Sprintf("rs-%s.qiniuapi.com", regionID), useHttps)}
	region.Rsf.Preferred = []string{makeHost(fmt.Sprintf("rsf-%s.qiniuapi.com", regionID), useHttps)}
	region.Api.Preferred = []string{makeHost(fmt.Sprintf("api-%s.qiniuapi.com", regionID), useHttps)}
	region.Bucket.Preferred = []string{makeHost("uc.qiniuapi.com", useHttps), makeHost("kodo-config.qiniuapi.com", useHttps)}
	region.Bucket.Alternative = []string{makeHost("uc.qbox.me", useHttps)}
	return region
}

func makeHost(domain string, useHttps bool) string {
	if useHttps {
		return "https://" + domain
	} else {
		return "http://" + domain
	}
}

func (r *Region) GetRegions(context.Context) ([]*Region, error) {
	return []*Region{r}, nil
}

func (r *Region) Endpoints(serviceNames []ServiceName) (Endpoints, error) {
	var endpoint Endpoints
	for _, serviceName := range serviceNames {
		switch serviceName {
		case ServiceUp:
			endpoint = endpoint.Join(r.Up)
		case ServiceIo:
			endpoint = endpoint.Join(r.Io)
		case ServiceIoSrc:
			endpoint = endpoint.Join(r.IoSrc)
		case ServiceRs:
			endpoint = endpoint.Join(r.Rs)
		case ServiceRsf:
			endpoint = endpoint.Join(r.Rsf)
		case ServiceApi:
			endpoint = endpoint.Join(r.Api)
		case ServiceBucket:
			endpoint = endpoint.Join(r.Bucket)
		default:
			return endpoint, ErrUnrecognizedServiceName
		}
	}
	return endpoint, nil
}

func (r *Region) EndpointsIter(serviceNames []ServiceName) (*EndpointsIter, error) {
	endpoints, err := r.Endpoints(serviceNames)
	if err != nil {
		return nil, err
	}
	return endpoints.Iter(), nil
}

func (r *Region) IsEqual(right *Region) bool {
	return r.RegionID == right.RegionID &&
		r.Up.IsEqual(right.Up) &&
		r.Io.IsEqual(right.Io) &&
		r.IoSrc.IsEqual(right.IoSrc) &&
		r.Rs.IsEqual(right.Rs) &&
		r.Rsf.IsEqual(right.Rsf) &&
		r.Api.IsEqual(right.Api) &&
		r.Bucket.IsEqual(right.Bucket)
}

func (ep Endpoints) Join(rights ...Endpoints) Endpoints {
	newEndpoint := ep
	for _, right := range rights {
		newEndpoint.Accelerated = append(newEndpoint.Accelerated, right.Accelerated...)
		newEndpoint.Preferred = append(newEndpoint.Preferred, right.Preferred...)
		newEndpoint.Alternative = append(newEndpoint.Alternative, right.Alternative...)
	}

	return newEndpoint
}

func (ep Endpoints) IsEqual(right Endpoints) bool {
	return reflect.DeepEqual(ep.Accelerated, right.Accelerated) &&
		reflect.DeepEqual(ep.Preferred, right.Preferred) &&
		reflect.DeepEqual(ep.Alternative, right.Alternative)

}

func (ep Endpoints) Iter() *EndpointsIter {
	return &EndpointsIter{endpoints: ep}
}

func (ep Endpoints) HostsLength() int {
	return len(ep.Accelerated) + len(ep.Preferred) + len(ep.Alternative)
}

func (ep Endpoints) IsEmpty() bool {
	return len(ep.Accelerated) == 0 && len(ep.Preferred) == 0 && len(ep.Alternative) == 0
}

func (ep Endpoints) firstUrl(useHttps bool) string {
	for _, accelerated := range ep.Accelerated {
		return makeUrlFromHost(accelerated, useHttps)
	}
	for _, preferred := range ep.Preferred {
		return makeUrlFromHost(preferred, useHttps)
	}
	for _, alternative := range ep.Alternative {
		return makeUrlFromHost(alternative, useHttps)
	}
	return ""
}

func (ep Endpoints) GetEndpoints(context.Context) (Endpoints, error) {
	return ep, nil
}

func (ep Endpoints) allUrls(useHttps bool) []string {
	allHosts := make([]string, 0, len(ep.Accelerated)+len(ep.Preferred)+len(ep.Alternative))
	for _, accelerated := range ep.Accelerated {
		allHosts = append(allHosts, makeUrlFromHost(accelerated, useHttps))
	}
	for _, preferred := range ep.Preferred {
		allHosts = append(allHosts, makeUrlFromHost(preferred, useHttps))
	}
	for _, alternative := range ep.Alternative {
		allHosts = append(allHosts, makeUrlFromHost(alternative, useHttps))
	}
	return allHosts
}

func (ep Endpoints) ToHostProvider() hostprovider.HostProvider {
	return &endpointsHostProvider{
		iter:    ep.Iter(),
		freezer: freezer.New(),
	}
}

func (ep Endpoints) Clone() Endpoints {
	return Endpoints{
		Preferred:   append([]string{}, ep.Preferred...),
		Alternative: append([]string{}, ep.Alternative...),
		Accelerated: append([]string{}, ep.Accelerated...),
	}
}

func makeUrlFromHost(host string, useHttps bool) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host
	}
	if useHttps {
		return "https://" + host
	} else {
		return "http://" + host
	}
}

func (iter *EndpointsIter) Next(nextHost *string) bool {
	for {
		switch iter.current {
		case endpointsStatusAccelerated:
			if iter.index >= len(iter.endpoints.Accelerated) {
				iter.current = endpointsStatusPreferred
				iter.index = 0
				continue
			}
			host := iter.endpoints.Accelerated[iter.index]
			iter.index += 1
			*nextHost = host
			return true
		case endpointsStatusPreferred:
			if iter.index >= len(iter.endpoints.Preferred) {
				iter.current = endpointsStatusAlternative
				iter.index = 0
				continue
			}
			host := iter.endpoints.Preferred[iter.index]
			iter.index += 1
			*nextHost = host
			return true
		case endpointsStatusAlternative:
			if iter.index >= len(iter.endpoints.Alternative) {
				return false
			}
			host := iter.endpoints.Alternative[iter.index]
			iter.index += 1
			*nextHost = host
			return true
		default:
			return false
		}
	}
}

func (iter *EndpointsIter) More() bool {
	switch iter.current {
	case endpointsStatusAccelerated:
		if iter.index >= len(iter.endpoints.Accelerated) {
			return len(iter.endpoints.Preferred) > 0
		}
		return true
	case endpointsStatusPreferred:
		if iter.index >= len(iter.endpoints.Preferred) {
			return len(iter.endpoints.Alternative) > 0
		}
		return true
	case endpointsStatusAlternative:
		return iter.index < len(iter.endpoints.Alternative)
	default:
		return false
	}
}

func (iter *EndpointsIter) SwitchToAlternative() {
	if len(iter.endpoints.Alternative) > 0 && iter.current != endpointsStatusAlternative {
		iter.current = endpointsStatusAlternative
		iter.index = 0
	}
}

func (iter *EndpointsIter) SwitchToPreferred() {
	if len(iter.endpoints.Preferred) > 0 && iter.current != endpointsStatusPreferred {
		iter.current = endpointsStatusPreferred
		iter.index = 0
	}
}

func (provider *endpointsHostProvider) Freeze(host string, cause error, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	provider.lastFreezeErr = cause
	return provider.freezer.Freeze(host, duration)
}

func (provider *endpointsHostProvider) Provider() (string, error) {
	if provider.iter.endpoints.IsEmpty() {
		return "", hostprovider.ErrNoHostFound
	}

	var host string
	for provider.iter.Next(&host) {
		if provider.freezer.Available(host) {
			return host, nil
		}
	}

	if provider.lastFreezeErr != nil {
		return "", provider.lastFreezeErr
	} else {
		return "", hostprovider.ErrAllHostsFrozen
	}
}
