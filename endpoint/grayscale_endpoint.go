package endpoint

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/endpoint"
	"grayscaleService/dataModels"
	"grayscaleService/server"
)

type GetAllModulesVresionsEndpointResponse struct {
	AllModules []dataModels.ModuleVersionRepsonse `json:"allModules"`
}

type CommonMsgResponse struct {
	Message string `json:'message'`
}

// 更新模块信息的参数
type UpdateDoModuleRequireStableRequest struct {
	ModuleId   int64 `json:"moduleId"`
	IsRequired int64 `json:"isRequired"`
}

// 更新版本稳定版本参数
type UpdateStableVersionRequest struct {
	ModuleId        int64 `json:"moduleId"`
	StableVersionId int64 `json:"stableVersionId"`
}

// 获取远程信息
type GetRemoteConfigureRequest struct {
	ModuleName string `json:"moduleName"`
	UserId     int64  `json:"userId"`
	IsServer   bool   `json:"isServer"`
}

type GetRemoteConfigureResponse struct {
	ModuleLink string `json:"moduleLink"`
}

type UploadResponse struct {
	Msg string `json:"msg"`
}

// 获取所有版本信息
func MakeGetAllModulesVresionsEndpoint(svc server.IGrayscaleModuleServive) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		modulesVersionsInfos, err := svc.GetAllModulesVresions(ctx)
		if err != nil {
			return nil, err
		}
		return GetAllModulesVresionsEndpointResponse{AllModules: modulesVersionsInfos}, nil
	}
}

// 更新模块信息
func MakeUpdateModuleEndpoint(svc server.IGrayscaleModuleServive) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dataModels.ModuleVersionReq)
		msg, err := svc.AddModuleInfo(ctx, req)
		if err != nil {
			return CommonMsgResponse{Message: msg}, err
		}
		return CommonMsgResponse{Message: msg}, nil
	}
}

// 设置 module 是否需要稳定版本
func MakeUpdateDoModuleRequireStableEndpoint(svc server.IGrayscaleModuleServive) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateDoModuleRequireStableRequest)
		msg, err := svc.UpdateDoModuleRequireStable(ctx, req.ModuleId, req.IsRequired)
		if err != nil {
			return CommonMsgResponse{err.Error()}, err
		}
		return CommonMsgResponse{Message: msg}, nil
	}
}

// 设置version 是否是稳定版本
func MakeUpdateStableVersionEndpoint(svc server.IGrayscaleModuleServive) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateStableVersionRequest)
		msg, err := svc.UpdateStableVersion(ctx, req.ModuleId, req.StableVersionId)
		if err != nil {
			return CommonMsgResponse{err.Error()}, err
		}
		return CommonMsgResponse{Message: msg}, nil
	}
}

// 获取远程资源地址
func MakeGetRemoteConfigureEndpoint(svc server.IGrayscaleModuleServive) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetRemoteConfigureRequest)
		jsonInfo, err := svc.GetRemoteConfigure(ctx, req.ModuleName, req.UserId, req.IsServer)
		if err != nil {
			fmt.Println("err")
			return jsonInfo, err
		}
		return jsonInfo, err
	}
}

// 上传(模块)文件资源
func MakeUploadFilesEndpoint(svc server.IGrayscaleModuleServive) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dataModels.FormData)
		// 对于模块/应用上传，先要更新数据库，信息， 再更新文件系统
		msg, err := svc.UploadFiles(ctx, req)
		if err != nil {
			fmt.Println("err")
			return UploadResponse{Msg: msg}, err
		}
		return UploadResponse{Msg: msg}, nil
	}
}
