package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/qiniu/go-sdk/v7/client"
	"io"
	"strings"
	"sync"
)

type UploadResumeVersion = int

const (
	UploadResumeV1 UploadResumeVersion = 1
	UploadResumeV2 UploadResumeVersion = 2

	uploadMethodForm     = 0
	uploadMethodResumeV1 = 1
	uploadMethodResumeV2 = 2
)

type UploadConfig struct {
	UseHTTPS      bool
	UseCdnDomains bool
	Regions       *RegionGroup
}

func (config *UploadConfig) init() {
}

type UploadExtra struct {
	// 【可选】参数，
	// 用户自定义参数，必须以 "x:" 开头。若不以 "x:" 开头，则忽略。
	// meta-data 参数，必须以 "x-qn-meta-" 开头。若不以 "x-qn-meta-" 开头，则忽略。
	Params map[string]string

	// 【可选】指定上传使用的 host, 如果指定则不再使用区域的 host
	UpHost string

	// 【可选】尝试次数
	TryTimes int

	// 【可选】主备域名冻结时间（单位：秒，默认：600），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	HostFreezeDuration int

	// 【可选】当为 "" 时候，服务端自动判断。
	MimeType string

	// 【可选】上传事件：进度通知。这个事件的回调函数应该尽可能快地结束。
	OnProgress func(fileSize, uploaded int64)

	// 【可选】分片上传的上传方式， 默认：UploadResumeV2
	UploadResumeVersion UploadResumeVersion

	// 【可选】上传阈值，当文件大小大于此阈值时使用分片上传；单位：字节，默认：4 * 1024 * 1024
	UploadThreshold int64

	// 【可选】分片上传进度记录
	Recorder Recorder

	// 【可选】分片上传时每次上传的块大小，单位：字节，默认：4 * 1024 * 1024
	PartSize int64
}

func (extra *UploadExtra) init() {
	if extra.TryTimes == 0 {
		extra.TryTimes = settings.TryTimes
	}
	if extra.HostFreezeDuration <= 0 {
		extra.HostFreezeDuration = 10 * 60
	}
	if extra.UploadResumeVersion != UploadResumeV1 {
		extra.UploadResumeVersion = UploadResumeV2
	}
	if extra.UploadThreshold <= 0 {
		extra.UploadThreshold = 4 * 1024 * 1024
	}
	if extra.PartSize <= 0 {
		extra.PartSize = 4 * 1024 * 1024
	}

	locker := sync.Mutex{}
	onProgress := extra.OnProgress
	uploadedSize := int64(0)
	extra.OnProgress = func(fileSize, uploaded int64) {
		if onProgress == nil {
			return
		}

		locker.Lock()
		if uploaded <= uploadedSize {
			locker.Unlock()
			return
		}
		uploadedSize = uploaded
		locker.Unlock()

		onProgress(fileSize, uploadedSize)
	}
}

func (extra *UploadExtra) getMetadata() map[string]string {
	if len(extra.Params) == 0 {
		return nil
	}

	ret := make(map[string]string)
	for key, value := range extra.Params {
		if strings.HasPrefix(key, "x-qn-meta-") && len(value) > 0 {
			ret[key] = value
		}
	}

	if len(ret) > 0 {
		return ret
	}

	return nil
}

func (extra *UploadExtra) getCustomVar() map[string]string {
	if len(extra.Params) == 0 {
		return nil
	}

	ret := make(map[string]string)
	for key, value := range extra.Params {
		if strings.HasPrefix(key, "x:") && len(value) > 0 {
			ret[key] = value
		}
	}

	if len(ret) > 0 {
		return ret
	}

	return nil
}

type UploadRet struct {
	Hash string `json:"hash"`
	Key  string `json:"key"`
}

type UploadManager struct {
	cfg    *UploadConfig
	client *client.Client
}

func NewUploadManager(cfg *UploadConfig) *UploadManager {
	return NewUploadManagerEx(cfg, nil)
}

func NewUploadManagerEx(cfg *UploadConfig, c *client.Client) *UploadManager {
	if cfg == nil {
		cfg = &UploadConfig{}
	}

	if c == nil {
		c = &client.DefaultClient
	}

	return &UploadManager{
		cfg:    cfg,
		client: c,
	}
}

func (manager *UploadManager) Put(ctx context.Context, ret interface{}, upToken string, key *string, source UploadSource, extra *UploadExtra) error {
	if ctx == nil {
		return errors.New("ctx can't be nil")
	}
	if ret == nil {
		return errors.New("ret invalid")
	}
	if len(upToken) == 0 {
		return errors.New("upToken invalid")
	}
	if source == nil {
		return errors.New("source invalid")
	}

	return manager.putRetryWithRegion(ctx, ret, upToken, key, source, extra)
}

func (manager *UploadManager) putRetryWithRegion(ctx context.Context, ret interface{}, upToken string, key *string, source UploadSource, extra *UploadExtra) error {
	if manager.cfg.Regions == nil {
		regions, err := manager.getRegionGroupWithUploadToken(upToken)
		if err != nil {
			return err
		}
		manager.cfg.Regions = regions
	}

	regions := manager.cfg.Regions.clone()

	resumeVersion := "v1"
	uploadMethod := uploadMethodForm
	if source.Size() > 0 && source.Size() < extra.UploadThreshold {
		uploadMethod = uploadMethodResumeV2
		resumeVersion = "v1"
	} else if extra.UploadResumeVersion == UploadResumeV1 {
		// 默认使用分片 v2，如果设置了 v1 则使用 v1
		uploadMethod = uploadMethodResumeV1
		resumeVersion = "v2"
	}

	if uploadMethod != uploadMethodForm {
		recoverRegion := manager.getRecoverRegion(key, upToken, resumeVersion, source, extra)
		if recoverRegion != nil {
			// 把记录的 Region 插在第一个
			regions.regions = append([]*Region{recoverRegion}, regions.regions...)
		}
	}

	var err error
	for {
		region := regions.GetRegion()
		err = manager.put(ctx, ret, region, uploadMethod, upToken, key, source, extra)

		// 是否需要重试
		if !shouldUploadRegionRetry(err) {
			break
		}

		// 切换区域是否成功
		if !regions.SwitchRegion() {
			break
		}
	}
	return err
}

func (manager *UploadManager) put(ctx context.Context, ret interface{}, region *Region, uploadMethod int,
	upToken string, key *string, source UploadSource, extra *UploadExtra) error {
	if extra == nil {
		extra = &UploadExtra{}
	}
	extra.init()

	if uploadMethod == uploadMethodForm {
		return manager.putByForm(ctx, ret, region, upToken, key, source, extra)
	} else if uploadMethod == uploadMethodResumeV1 {
		return manager.putByResumeV1(ctx, ret, region, upToken, key, source, extra)
	} else {
		return manager.putByResumeV2(ctx, ret, region, upToken, key, source, extra)
	}
}

func (manager *UploadManager) putByForm(ctx context.Context, ret interface{}, region *Region, upToken string, key *string, source UploadSource, extra *UploadExtra) error {
	saveKey, hasKey := uploadKey(key)
	uploadExtra := &PutExtra{
		Params:             extra.Params,
		UpHost:             extra.UpHost,
		TryTimes:           extra.TryTimes,
		HostFreezeDuration: extra.HostFreezeDuration,
		MimeType:           extra.MimeType,
		OnProgress:         extra.OnProgress,
	}
	uploader := manager.getFormUploader(region)

	if reader, ok := source.(*uploadSourceReader); ok {
		return uploader.put(ctx, ret, upToken, saveKey, hasKey, reader.reader, -1, uploadExtra, "")
	}

	if readerAt, ok := source.(*uploadSourceReaderAt); ok {
		reader := io.NewSectionReader(readerAt.reader, 0, readerAt.size)
		return uploader.put(ctx, ret, upToken, saveKey, hasKey, reader, readerAt.size, uploadExtra, "")
	}

	if reader, ok := source.(*uploadSourceFile); ok {
		return uploader.putFile(ctx, ret, upToken, saveKey, hasKey, reader.filePath, uploadExtra)
	}

	return errors.New("unknown upload source")
}

func (manager *UploadManager) putByResumeV1(ctx context.Context, ret interface{}, region *Region, upToken string, key *string, source UploadSource, extra *UploadExtra) error {
	locker := sync.Mutex{}
	uploadedSize := int64(0)
	saveKey, hasKey := uploadKey(key)
	uploadExtra := &RputExtra{
		Recorder:           extra.Recorder,
		Params:             extra.Params,
		UpHost:             extra.UpHost,
		MimeType:           extra.MimeType,
		ChunkSize:          int(extra.PartSize),
		TryTimes:           extra.TryTimes,
		HostFreezeDuration: extra.HostFreezeDuration,
		Progresses:         nil,
		Notify: func(blkIdx int, blkSize int, ret *BlkputRet) {
			if blkIdx < 2 {
				return
			}
			locker.Lock()
			offset := int64(blkIdx-1) * int64(blkSize)
			if offset > uploadedSize {
				uploadedSize = offset
			}
			locker.Unlock()
			extra.OnProgress(source.Size(), uploadedSize)
		},
		NotifyErr: nil,
	}
	uploader := manager.getResumeV1Uploader(region)

	if reader, ok := source.(*uploadSourceReader); ok {
		return uploader.rputWithoutSize(ctx, ret, upToken, saveKey, hasKey, reader.reader, uploadExtra)
	}

	if reader, ok := source.(*uploadSourceReaderAt); ok {
		return uploader.rput(ctx, ret, upToken, saveKey, hasKey, reader.reader, reader.size, nil, uploadExtra)
	}

	if reader, ok := source.(*uploadSourceFile); ok {
		return uploader.rputFile(ctx, ret, upToken, saveKey, hasKey, reader.filePath, uploadExtra)
	}

	return errors.New("unknown upload source")
}

func (manager *UploadManager) putByResumeV2(ctx context.Context, ret interface{}, region *Region, upToken string, key *string, source UploadSource, extra *UploadExtra) error {
	locker := sync.Mutex{}
	uploadedSize := int64(0)
	saveKey, hasKey := uploadKey(key)
	uploadExtra := &RputV2Extra{
		Recorder:           extra.Recorder,
		Metadata:           extra.getMetadata(),
		CustomVars:         extra.getCustomVar(),
		UpHost:             extra.UpHost,
		MimeType:           extra.MimeType,
		PartSize:           extra.PartSize,
		TryTimes:           extra.TryTimes,
		HostFreezeDuration: extra.HostFreezeDuration,
		Progresses:         nil,
		Notify: func(partNumber int64, ret *UploadPartsRet) {
			if partNumber < 2 {
				return
			}
			locker.Lock()
			offset := (partNumber - 1) * extra.PartSize
			if offset > uploadedSize {
				uploadedSize = offset
			}
			locker.Unlock()
			extra.OnProgress(source.Size(), uploadedSize)
		},
		NotifyErr: nil,
	}
	uploader := manager.getResumeV2Uploader(region)

	if reader, ok := source.(*uploadSourceReader); ok {
		return uploader.rputWithoutSize(ctx, ret, upToken, saveKey, hasKey, reader.reader, uploadExtra)
	}

	if reader, ok := source.(*uploadSourceReaderAt); ok {
		return uploader.rput(ctx, ret, upToken, saveKey, hasKey, reader.reader, reader.size, nil, uploadExtra)
	}

	if reader, ok := source.(*uploadSourceFile); ok {
		return uploader.rputFile(ctx, ret, upToken, saveKey, hasKey, reader.filePath, uploadExtra)
	}

	return errors.New("unknown upload source")
}

func (manager *UploadManager) getRecoverRegion(key *string, upToken string, resumeVersion string,
	source UploadSource, extra *UploadExtra) *Region {
	file, ok := source.(*uploadSourceFile)
	if !ok {
		return nil
	}

	saveKey, hasKey := uploadKey(key)
	if !hasKey {
		return nil
	}

	recorderKey := getRecorderKey(extra.Recorder, upToken, saveKey, resumeVersion, extra.PartSize, &fileDetailsInfo{
		fileFullPath: file.filePath,
		fileInfo:     file.fileInfo,
	})
	if len(recorderKey) == 0 {
		return nil
	}

	recoverData, err := extra.Recorder.Get(recorderKey)
	if err != nil {
		return nil
	}

	var recoveryInfo uploaderRecoveryInfo
	if err := json.Unmarshal(recoverData, &recoveryInfo); err != nil {
		return nil
	}

	return recoveryInfo.Region
}

func (manager *UploadManager) getFormUploader(region *Region) *FormUploader {
	return NewFormUploaderEx(&Config{
		Zone:          region,
		Region:        region,
		UseHTTPS:      manager.cfg.UseHTTPS,
		UseCdnDomains: manager.cfg.UseCdnDomains,
		CentralRsHost: "",
	}, manager.client)
}

func (manager *UploadManager) getResumeV1Uploader(region *Region) *ResumeUploader {
	return NewResumeUploaderEx(&Config{
		Zone:          region,
		Region:        region,
		UseHTTPS:      manager.cfg.UseHTTPS,
		UseCdnDomains: manager.cfg.UseCdnDomains,
		CentralRsHost: "",
	}, manager.client)
}

func (manager *UploadManager) getResumeV2Uploader(region *Region) *ResumeUploaderV2 {
	return NewResumeUploaderV2Ex(&Config{
		Zone:          region,
		Region:        region,
		UseHTTPS:      manager.cfg.UseHTTPS,
		UseCdnDomains: manager.cfg.UseCdnDomains,
		CentralRsHost: "",
	}, manager.client)
}

func (manager *UploadManager) getRegionGroupWithUploadToken(upToken string) (*RegionGroup, error) {
	ak, bucket, err := getAkBucketFromUploadToken(upToken)
	if err != nil {
		return nil, err
	}
	return getRegionGroup(ak, bucket)
}

func uploadKey(keyQuote *string) (key string, hashKey bool) {
	if keyQuote == nil {
		return "", false
	} else {
		return *keyQuote, true
	}
}

type uploaderRecoveryInfo struct {
	Region *Region `json:"r"`
}
