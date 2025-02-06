package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
	RawData []byte                 `json:"raw_data"`
}

func (r Response) IsSuccess() bool {
	return r.Status == "success"
}

func (r Response) GetString(key string) string {
	if val, ok := r.Data[key].(string); ok {
		return val
	}
	return ""
}

func MakeRequest(method, path string, body interface{}, token string) Response {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return Response{Status: "error", Message: fmt.Sprintf("Failed to marshal request body: %v", err)}
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, apiURL+path, reqBody)
	if err != nil {
		return Response{Status: "error", Message: fmt.Sprintf("Failed to create request: %v", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Response{Status: "error", Message: fmt.Sprintf("Request failed: %v", err)}
	}
	defer resp.Body.Close()

	var response Response
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return Response{Status: "error", Message: fmt.Sprintf("Failed to decode response: %v", err)}
	}

	return response
}
