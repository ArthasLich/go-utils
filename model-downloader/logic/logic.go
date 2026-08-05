package logic

import (
	"fmt"
	"net/http"
)

type innerData struct {
	StorageSize uint64 `json:"StorageSize"`
}

type QuerySuccessResult struct {
	Code      int        `json:"Code"`
	Data      *innerData `json:"Data,omitempty"`
	Message   string     `json:"Message"`
	RequestId string     `json:"RequestId"`
	Success   bool       `json:"Success"`
}

// QueryModelSize 查询模型大小，单位是字节
func QueryModelSize(model string) (uint64, error) {
	result := QuerySuccessResult{}
	_, err := HTTPClient.R().SetPathParam("model", model).SetResult(&result).SetError(&result).Get("https://modelscope.cn/api/v1/models/{model}")
	if err != nil {
		return 0, err
	}
	if result.Code == http.StatusOK && result.Success {
		return result.Data.StorageSize, nil
	}
	return 0, fmt.Errorf("error: get model size failed: %s", result.Message)
}
