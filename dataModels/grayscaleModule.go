package dataModels

import "mime/multipart"

type Module struct {
	Id              int64  `json:"id" sql:"id" req:"id"`
	ModuleName      string `json:"moduleName" sql:"moduleName" req:"moduleName"`
	IsUseValid      int64  `json:"isUseValid" sql:"isUseValid" req:"isUseValid"` // 是否使用稳定版本
	LatestVersionId int64  `json:"latestVersionId" sql:"latestVersionId" req:"latestVersionId"`
}

type Version struct {
	Id       int64  `json:"id" sql:"id" req:"id"`
	PId      int64  `json:"pid" sql:"pid" req:"pid"`
	Version  string `json:"version" sql:"version" req:"version"`
	IsStable int64  `json:"isStable" sql:"isStable" req:"isStable"`
}

// ModuleAndVersion from sql
type ModuleVersionSql struct {
	ModuleId        int64  `json:"moduleId" sql:"moduleId" req:"moduleId"`
	ModuleName      string `json:"moduleName" sql:"moduleName" req:"moduleName"`
	IsUseValid      int64  `json:"isUseValid" sql:"isUseValid" req:"isUseValid"` // 是否使用稳定版本
	LatestVersionId int64  `json:"latestVersionId" sql:"latestVersionId" req:"latestVersionId"`
	VersionId       int64  `json:"versionId" sql:"versionId" req:"versionId"`
	Version         string `json:"version" sql:"version" req:"version"`
	IsStable        int64  `json:"isStable" sql:"isStable" req:"isStable"`
}

// 更新模块+版本参数 module + version
type ModuleVersionReq struct {
	ModuleName string `json:"moduleName"`
	IsUseValid int64  `json:"isUseValid"` // 是否使用稳定版本
	Version    string `json:"version"`
	IsStable   int64  `json:"isStable"`
}

type VersionResponse struct {
	VersionId int64  `json:"versionId" sql:"versionId" req:"versionId"`
	Version   string `json:"version" sql:"version" req:"version"`
	IsStable  int64  `json:"isStable" sql:"isStable" req:"isStable"`
}

type ModuleVersionRepsonse struct {
	ModuleId        int64              `json:"moduleId" sql:"moduleId" req:"moduleId"`
	ModuleName      string             `json:"moduleName" sql:"moduleName" req:"moduleName"`
	IsUseValid      int64              `json:"isUseValid" sql:"isUseValid" req:"isUseValid"` // 是否使用稳定版本
	LatestVersionId int64              `json:"latestVersionId" sql:"latestVersionId" req:"latestVersionId"`
	VersionList     []*VersionResponse `json:"versionList"`
}

// 文件夹(前端资源文件)上传
// 资源类型结构体
type uploadFiles struct {
	//Assets Folder `json:"assets"`
	Assets []byte `json:"assets"`
}
type FormData struct {
	AssetsHeader *multipart.FileHeader `form:"assetsHeader"`
	Assets       multipart.File        `form:"assets"`
	IsUseValid   int64                 `json:"isUseValid"`
}

type Folder struct {
	Name    string
	Files   []*multipart.FileHeader
	Folders []Folder
}

// 元数据
type MetaData struct {
	App MetaDataAppInfo `json:"app"`
}

type MetaDataAppInfo struct {
	Name    string `json:"name"`
	Version string `json:"build_version"`
}
