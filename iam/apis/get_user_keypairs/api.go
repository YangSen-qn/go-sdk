// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 列举 IAM 子账号密钥
package get_user_keypairs

import (
	"encoding/json"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
)

// 调用 API 所用的请求
type Request struct {
	Alias       string                          // 子账号别名
	Page        int64                           // 分页页号，从 1 开始，默认 1
	PageSize    int64                           // 分页大小，默认 20，最大 2000
	Credentials credentials.CredentialsProvider // 鉴权参数，用于生成鉴权凭证，如果为空，则使用 HTTPClientOptions 中的 CredentialsProvider
}

// 获取 API 所用的响应
type Response struct {
	Data GetIamUserKeyPairsData // IAM 子账号密钥信息
}

// 返回的 IAM 子账号密钥
type GetIamUserKeyPair struct {
	Id        string // 记录 ID
	AccessKey string // IAM 子账号 Access Key
	SecretKey string // IAM 子账号 Secret Key
	UserId    string // 关联用户的记录 ID
	CreatedAt string // 密钥创建时间
	Enabled   bool   // 密钥是否启用
}
type jsonGetIamUserKeyPair struct {
	Id        string `json:"id"`         // 记录 ID
	AccessKey string `json:"access_key"` // IAM 子账号 Access Key
	SecretKey string `json:"secret_key"` // IAM 子账号 Secret Key
	UserId    string `json:"user_id"`    // 关联用户的记录 ID
	CreatedAt string `json:"created_at"` // 密钥创建时间
	Enabled   bool   `json:"enabled"`    // 密钥是否启用
}

func (j *GetIamUserKeyPair) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonGetIamUserKeyPair{Id: j.Id, AccessKey: j.AccessKey, SecretKey: j.SecretKey, UserId: j.UserId, CreatedAt: j.CreatedAt, Enabled: j.Enabled})
}
func (j *GetIamUserKeyPair) UnmarshalJSON(data []byte) error {
	var nj jsonGetIamUserKeyPair
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.Id = nj.Id
	j.AccessKey = nj.AccessKey
	j.SecretKey = nj.SecretKey
	j.UserId = nj.UserId
	j.CreatedAt = nj.CreatedAt
	j.Enabled = nj.Enabled
	return nil
}
func (j *GetIamUserKeyPair) validate() error {
	if j.Id == "" {
		return errors.MissingRequiredFieldError{Name: "Id"}
	}
	if j.AccessKey == "" {
		return errors.MissingRequiredFieldError{Name: "AccessKey"}
	}
	if j.SecretKey == "" {
		return errors.MissingRequiredFieldError{Name: "SecretKey"}
	}
	if j.UserId == "" {
		return errors.MissingRequiredFieldError{Name: "UserId"}
	}
	if j.CreatedAt == "" {
		return errors.MissingRequiredFieldError{Name: "CreatedAt"}
	}
	return nil
}

// 返回的 IAM 子账号密钥列表
type GetIamUserKeyPairs = []GetIamUserKeyPair

// IAM 子账号密钥信息
type Data struct {
	Count int64              // IAM 子账号密钥数量
	List  GetIamUserKeyPairs // IAM 子账号密钥列表
}

// 返回的 IAM 子账号密钥列表信息
type GetIamUserKeyPairsData = Data
type jsonData struct {
	Count int64              `json:"count"` // IAM 子账号密钥数量
	List  GetIamUserKeyPairs `json:"list"`  // IAM 子账号密钥列表
}

func (j *Data) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonData{Count: j.Count, List: j.List})
}
func (j *Data) UnmarshalJSON(data []byte) error {
	var nj jsonData
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.Count = nj.Count
	j.List = nj.List
	return nil
}
func (j *Data) validate() error {
	if j.Count == 0 {
		return errors.MissingRequiredFieldError{Name: "Count"}
	}
	if len(j.List) == 0 {
		return errors.MissingRequiredFieldError{Name: "List"}
	}
	for _, value := range j.List {
		if err := value.validate(); err != nil {
			return err
		}
	}
	return nil
}

// 返回的 IAM 子账号密钥列表响应
type GetIamUserKeyPairsResp = Response
type jsonResponse struct {
	Data GetIamUserKeyPairsData `json:"data"` // IAM 子账号密钥信息
}

func (j *Response) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonResponse{Data: j.Data})
}
func (j *Response) UnmarshalJSON(data []byte) error {
	var nj jsonResponse
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.Data = nj.Data
	return nil
}
func (j *Response) validate() error {
	if err := j.Data.validate(); err != nil {
		return err
	}
	return nil
}