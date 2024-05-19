package http

import (
	"context"
	"encoding/json"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"grayscaleService/dataModels"
	"grayscaleService/endpoint"
	"grayscaleService/server"
	"net/http"
	"strconv"
)

func decodeGetAllModulesVresionsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

// 更新module和version信息
func decodeUpdateModuleInfoRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dataModels.ModuleVersionReq
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

// 更新module 是否设置稳定版本
func decodeUpdateDoModuleRequireStableRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request endpoint.UpdateDoModuleRequireStableRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

// 更新version 是否是稳定版本
func decodeUpdateStableVersionRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request endpoint.UpdateStableVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

// 获取远程信息模块链接
func decodeGetRemoteConfigureRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request endpoint.GetRemoteConfigureRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

// 上传(模块)文件 (.tar.gz的压缩文件)
func decodeUploadFilesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dataModels.FormData
	//err := r.ParseMultipartForm(32 << 20) // 32 MB
	//if err != nil {
	//	return nil, err
	//}
	//// Decode the JSON data in the "json_data" field, if it exists
	//jsonStr := r.PostFormValue("assets")
	//if jsonStr != "" {
	//	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	//	err = decoder.Decode(&request)
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	// gizp post
	//gzipReader, err := gzip.NewReader(r.Body)
	//if err != nil {
	//	return nil, err
	//}
	//defer gzipReader.Close()
	//buf := new(bytes.Buffer)
	//_, err = io.Copy(buf, gzipReader)
	//if err != nil {
	//	return nil, err
	//}
	//request.Assets = buf.Bytes()
	// formdata post
	err := r.ParseMultipartForm(32 << 20) // 32MB大小限制
	if err != nil {
		return nil, err
	}
	request.Assets, request.AssetsHeader, err = r.FormFile("assets")
	request.IsUseValid, err = strconv.ParseInt(r.FormValue("isUseValid"), 10, 64)
	if err != nil {
		return nil, err
	}
	return request, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

func NewHTTPServer(svc server.IGrayscaleModuleServive) http.Handler {
	GetAllModulesVresionsHandle := httptransport.NewServer(
		endpoint.MakeGetAllModulesVresionsEndpoint(svc),
		decodeGetAllModulesVresionsRequest,
		encodeResponse,
	)
	UpdateModuleInfoHandle := httptransport.NewServer(
		endpoint.MakeUpdateModuleEndpoint(svc),
		decodeUpdateModuleInfoRequest,
		encodeResponse,
	)
	UpdateDoModuleRequireStableHandle := httptransport.NewServer(
		endpoint.MakeUpdateDoModuleRequireStableEndpoint(svc),
		decodeUpdateDoModuleRequireStableRequest,
		encodeResponse,
	)
	UpdateStableVersionHandle := httptransport.NewServer(
		endpoint.MakeUpdateStableVersionEndpoint(svc),
		decodeUpdateStableVersionRequest,
		encodeResponse,
	)
	GetRemoteConfigureHandle := httptransport.NewServer(
		endpoint.MakeGetRemoteConfigureEndpoint(svc),
		decodeGetRemoteConfigureRequest,
		encodeResponse,
	)
	UploadFilesHandle := httptransport.NewServer(
		endpoint.MakeUploadFilesEndpoint(svc),
		decodeUploadFilesRequest,
		encodeResponse,
	)

	r := mux.NewRouter()
	r.Handle("/getAllModues", GetAllModulesVresionsHandle).Methods("GET")
	r.Handle("/updateModule", UpdateModuleInfoHandle).Methods("POST")
	r.Handle("/updateModuleIsStable", UpdateDoModuleRequireStableHandle).Methods("POST")
	r.Handle("/updateStableVersion", UpdateStableVersionHandle).Methods("POST")
	r.Handle("/getRemoteConfigure", GetRemoteConfigureHandle).Methods("POST")
	r.Handle("/uploadFiles", UploadFilesHandle).Methods("POST")
	return r
}
