//go:build integration
// +build integration

package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestResumeUploadPutFile(t *testing.T) {
	var putRet PutRet
	ctx := context.TODO()
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testKey := fmt.Sprintf("testRPutFileKey_%d", r.Int())

	// prepare file for test uploading
	testLocalFile, err := ioutil.TempFile("", "TestResumeUploadPutFile")
	if err != nil {
		t.Fatalf("ioutil.TempFile file failed, err: %v", err)
	}
	defer os.Remove(testLocalFile.Name())

	err = resumeUploader.PutFile(ctx, &putRet, upToken, testKey, testLocalFile.Name(), nil)
	if err != nil {
		t.Fatalf("ResumeUploader#PutFile() error, %s", err)
	}
	t.Logf("Key: %s, Hash:%s", putRet.Key, putRet.Hash)
}

func TestResumeUploadWithInvalidUpHost(t *testing.T) {
	var putRet PutRet
	ctx := context.TODO()
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testKey := fmt.Sprintf("testRPutFileKey_%d", r.Int())

	// prepare file for test uploading
	testLocalFile, err := ioutil.TempFile("", "TestResumeUploadPutFile")
	if err != nil {
		t.Fatalf("ioutil.TempFile file failed, err: %v", err)
	}
	defer os.Remove(testLocalFile.Name())

	err = resumeUploader.PutFile(ctx, &putRet, upToken, testKey, testLocalFile.Name(), &RputExtra{
		UpHost: "mock.upload.io",
	})
	if err == nil {
		t.Fatalf("TestFormUploaderWithInvalidUpHost should error, %s", err)
	}
	if !strings.Contains(err.Error(), "lookup mock.upload.io: no such host") {
		t.Fatalf("TestFormUploaderWithInvalidUpHost should use mock host, %s", err)
	}
}

func TestPutWithoutSize(t *testing.T) {
	var putRet PutRet
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sizes := []int64{
		64,
		1 << blockBits,
		(1 << blockBits) - 1,
		(1 << blockBits) + 1,
		(1 << (blockBits + 4)) + 1,
	}
	chunkSizes := []int{0, 1 << 20}

	for _, chunkSize := range chunkSizes {
		for _, size := range sizes {
			md5Sumer := md5.New()
			rd := io.TeeReader(&io.LimitedReader{R: r, N: size}, md5Sumer)
			testKey := fmt.Sprintf("testRPutFileKey_%d", r.Int())
			err := resumeUploader.PutWithoutSize(context.Background(), &putRet, upToken, testKey, rd, &RputExtra{
				ChunkSize: chunkSize,
				Notify: func(blkIdx int, blkSize int, ret *BlkputRet) {
					t.Logf("Notify: blkIdx: %d, blkSize: %d, ret: %#v", blkIdx, blkSize, ret)
				},
				NotifyErr: func(blkIdx int, blkSize int, err error) {
					t.Logf("NotifyErr: blkIdx: %d, blkSize: %d, err: %s", blkIdx, blkSize, err)
				},
			})
			if err != nil {
				t.Fatalf("PutWithoutSize() error, %s", err)
			}
			md5Value := hex.EncodeToString(md5Sumer.Sum(nil))
			validateMD5(t, testKey, md5Value, size)
			t.Logf("Size: %d, Chunk: %d, Key: %s, Hash:%s", size, chunkSize, putRet.Key, putRet.Hash)
		}
	}
}

func TestPutWithSize(t *testing.T) {
	var putRet PutRet
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sizes := []int64{
		64,
		1 << blockBits,
		(1 << blockBits) - 1,
		(1 << blockBits) + 1,
		(1 << (blockBits + 4)) + 1,
	}
	chunkSizes := []int{0, 1 << 20}

	hostSelector := 0
	for _, chunkSize := range chunkSizes {
		hostSelector += 1
		for _, size := range sizes {
			hostSelector += 1
			upHost := ""
			if hostSelector%3 == 1 {
				upHost = testUpHost
			} else {
				upHost = "https://" + testUpHost
			}

			data := make([]byte, size)
			if _, err := io.ReadFull(r, data); err != nil {
				t.Fatal(err)
			}
			testKey := fmt.Sprintf("testRPutFileKey_%d", r.Int())
			err := resumeUploader.Put(context.Background(), &putRet, upToken, testKey, bytes.NewReader(data), size, &RputExtra{
				ChunkSize: chunkSize,
				UpHost:    upHost,
				Notify: func(blkIdx int, blkSize int, ret *BlkputRet) {
					t.Logf("Notify: blkIdx: %d, blkSize: %d, ret: %#v", blkIdx, blkSize, ret)
				},
				NotifyErr: func(blkIdx int, blkSize int, err error) {
					t.Logf("NotifyErr: blkIdx: %d, blkSize: %d, err: %s", blkIdx, blkSize, err)
				},
			})
			if err != nil {
				t.Fatalf("Put() error, %s", err)
			}
			md5ByteArray := md5.Sum(data)
			md5Value := hex.EncodeToString(md5ByteArray[:])
			validateMD5(t, testKey, md5Value, size)
			t.Logf("Size: %d, Chunk: %d, Key: %s, Hash:%s", size, chunkSize, putRet.Key, putRet.Hash)
		}
	}
}

func TestPutWithoutSizeAndKey(t *testing.T) {
	var putRet PutRet
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sizes := []int64{
		64,
		1 << blockBits,
		(1 << blockBits) - 1,
		(1 << blockBits) + 1,
		(1 << (blockBits + 4)) + 1,
	}
	chunkSizes := []int{0, 1 << 20}

	for _, chunkSize := range chunkSizes {
		for _, size := range sizes {
			md5Sumer := md5.New()
			rd := io.TeeReader(&io.LimitedReader{R: r, N: size}, md5Sumer)
			err := resumeUploader.PutWithoutKeyAndSize(context.Background(), &putRet, upToken, rd, &RputExtra{
				ChunkSize: chunkSize,
				Notify: func(blkIdx int, blkSize int, ret *BlkputRet) {
					t.Logf("Notify: blkIdx: %d, blkSize: %d, ret: %#v", blkIdx, blkSize, ret)
				},
				NotifyErr: func(blkIdx int, blkSize int, err error) {
					t.Logf("NotifyErr: blkIdx: %d, blkSize: %d, err: %s", blkIdx, blkSize, err)
				},
			})
			if err != nil {
				t.Fatalf("PutWithoutSize() error, %s", err)
			}
			testKey := putRet.Key
			md5Value := hex.EncodeToString(md5Sumer.Sum(nil))
			validateMD5(t, testKey, md5Value, size)
			t.Logf("Size: %d, Chunk: %d, Key: %s, Hash:%s", size, chunkSize, putRet.Key, putRet.Hash)
		}
	}
}

func TestPutWithRecovery(t *testing.T) {
	var putRet PutRet
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testKey := fmt.Sprintf("testRPutFileKey_%d", r.Int())
	dirName, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirName)
	recorder, err := NewFileRecorder(dirName)
	if err != nil {
		t.Fatal(err)
	}

	fileName := filepath.Join(dirName, "originalFile")
	testFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer testFile.Close()

	size := int64(4 * (1 << blockBits))
	io.CopyN(testFile, r, size)

	for i := 0; i < 4; i++ {
		ctx, cancelFunc := context.WithCancel(context.Background())
		counter := uint32(0)
		err = resumeUploader.PutFile(ctx, &putRet, upToken, testKey, fileName, &RputExtra{
			Recorder:  recorder,
			ChunkSize: (1 << blockBits) / 2,
			Notify: func(blkIdx int, blkSize int, ret *BlkputRet) {
				t.Logf("[%d] Notify: blkIdx: %d, blkSize: %d, ret: %#v", i, blkIdx, blkSize, ret)
				if atomic.AddUint32(&counter, 1) >= 2 {
					cancelFunc()
				}
			},
			NotifyErr: func(blkIdx int, blkSize int, err error) {
				t.Logf("[%d] NotifyErr: blkIdx: %d, blkSize: %d, err: %s", i, blkIdx, blkSize, err)
			},
		})
		if err == nil {
			return
		}
	}
	t.Fatal(err)
}

func TestPutWithBackup(t *testing.T) {
	var putRet PutRet
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	sizes := []int64{
		64,
		1 << blockBits,
		(1 << blockBits) - 1,
		(1 << blockBits) + 1,
		(1 << (blockBits + 4)) + 1,
	}
	chunkSizes := []int{0, 1 << 20}

	region, err := GetRegion(mac.AccessKey, testBucket)
	if err != nil {
		t.Fatal("get region error:", err)
	}

	customizedHost := []string{"mock.qiniu.com"}
	customizedHost = append(customizedHost, region.SrcUpHosts...)
	region.SrcUpHosts = customizedHost
	cfg := Config{}
	cfg.UseCdnDomains = false
	cfg.Region = region
	uploader := NewResumeUploader(&cfg)

	hostSelector := 0
	for _, chunkSize := range chunkSizes {
		hostSelector += 1
		for _, size := range sizes {
			hostSelector += 1
			data := make([]byte, size)
			if _, err := io.ReadFull(r, data); err != nil {
				t.Fatal(err)
			}
			testKey := fmt.Sprintf("testRPutFileKey_%d", r.Int())
			err := uploader.Put(context.Background(), &putRet, upToken, testKey, bytes.NewReader(data), size, &RputExtra{
				ChunkSize: chunkSize,
				Notify: func(blkIdx int, blkSize int, ret *BlkputRet) {
					t.Logf("Notify: blkIdx: %d, blkSize: %d, ret: %#v", blkIdx, blkSize, ret)
				},
				NotifyErr: func(blkIdx int, blkSize int, err error) {
					t.Logf("NotifyErr: blkIdx: %d, blkSize: %d, err: %s", blkIdx, blkSize, err)
				},
			})
			if err != nil {
				t.Fatalf("Put() error, %s", err)
			}
			md5ByteArray := md5.Sum(data)
			md5Value := hex.EncodeToString(md5ByteArray[:])
			validateMD5(t, testKey, md5Value, size)
			t.Logf("Size: %d, Chunk: %d, Key: %s, Hash:%s", size, chunkSize, putRet.Key, putRet.Hash)
		}
	}
}

func validateMD5(t *testing.T, key, md5Expected string, sizeExpected int64) {
	var (
		body struct {
			Hash  string `json:"hash"`
			Fsize int64  `json:"fsize"`
		}
		httpClient http.Client
	)

	response, err := httpClient.Get("http://" + testBucketDomain + "/" + key + "?qhash/md5")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("MD5 Get Status code is not ok, expected: %d, actual: %d, req_id: %s", http.StatusOK, response.StatusCode, response.Header.Get("X-Reqid"))
	}
	if err = json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Hash != md5Expected {
		t.Fatalf("MD5 Dismatch, expected: %s, actual: %s", md5Expected, body.Hash)
	}
	if body.Fsize != sizeExpected {
		t.Fatalf("File Size Dismatch, expected: %d, actual: %d", sizeExpected, body.Fsize)
	}
}

func TestRetryChecker(t *testing.T) {
	var putRet PutRet
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}

	upToken := putPolicy.UploadToken(mac)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testKey := fmt.Sprintf("testRPutFileKey_%d", r.Int())

	mycfg := Config{}
	wrongZone := Region{
		SrcUpHosts: []string{
			"nocname-up.qiniup.com",
			"nocname-up-nb.qiniup.com",
			"nocname-up-xs.qiniup.com",
		},
		CdnUpHosts: []string{
			"nocname-upload.qiniup.com",
			"nocname-upload-nb.qiniup.com",
			"nocname-upload-xs.qiniup.com",
		},
		RsHost:    "rs.qbox.me",
		RsfHost:   "rsf.qbox.me",
		ApiHost:   "api.qiniu.com",
		IovipHost: "iovip.qbox.me",
	}
	mycfg.Zone = &wrongZone
	mycfg.UseCdnDomains = true
	myResumeUploader := NewResumeUploaderEx(&mycfg, &clt)

	rd := strings.NewReader("hello world")
	// host unkown, so go to retry,
	// any way, no : panic: runtime error: invalid memory address or nil pointer dereference
	err := myResumeUploader.PutWithoutSize(context.Background(), &putRet, upToken, testKey, rd, nil)
	if err != nil {
		t.Logf("TestRetryChecker() error, %s", err)
	} else {
		t.Fatalf("TestRetryChecker() should failed, %s", putRet)
	}
}
