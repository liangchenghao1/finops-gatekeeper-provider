package utls

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
)

// ReadProviderRequest 读取并解析ProviderRequest
func ReadProviderRequest(w http.ResponseWriter, req *http.Request) (*externaldata.ProviderRequest, error) {
	// 只接受POST请求
	if req.Method != http.MethodPost {
		SendResponse(w, nil, "only POST is allowed")
		return nil, fmt.Errorf("only POST is allowed")
	}

	// 读取请求体
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		SendResponse(w, nil, fmt.Sprintf("unable to read request body: %v", err))
		return nil, err
	}

	// 解析请求体
	var providerRequest externaldata.ProviderRequest
	if err := json.Unmarshal(requestBody, &providerRequest); err != nil {
		SendResponse(w, nil, fmt.Sprintf("unable to unmarshal request body: %v", err))
		return nil, err
	}

	return &providerRequest, nil
}

// SendResponse 发送响应给Gatekeeper
func SendResponse(w http.ResponseWriter, results *[]externaldata.Item, systemErr string) {
	response := externaldata.ProviderResponse{
		APIVersion: "externaldata.gatekeeper.sh/v1alpha1",
		Kind:       "ProviderResponse",
		Response: externaldata.Response{
			Idempotent: true,
		},
	}

	if results != nil {
		response.Response.Items = *results
	} else {
		response.Response.SystemError = systemErr
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

func ParseWorkloadKey(key string) (kind, namespace, name string, err error) {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("invalid workload key %q, expect <namespace>/<name>", key)
	}
	return parts[0], parts[1], parts[2], nil
}
