package server

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mbobakov/grpc-consul-resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"grayscaleService/dataModels"
	pb "grayscaleService/pb/user"
	"grayscaleService/repositories"
	"grayscaleService/util"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type IGrayscaleModuleServive interface {
	GetAllModulesVresions(context.Context) ([]dataModels.ModuleVersionRepsonse, error)
	AddModuleInfo(context.Context, dataModels.ModuleVersionReq) (string, error)
	UpdateDoModuleRequireStable(context.Context, int64, int64) (string, error)
	UpdateStableVersion(context.Context, int64, int64) (string, error)
	GetRemoteConfigure(context.Context, string, int64, bool) (interface{}, error)
	UploadFiles(context.Context, dataModels.FormData) (string, error)
}

type GrayscaleModuleServive struct {
	grayscaleRepositories repositories.IGrayscaleManager
	consulAddress         string
	minioClient           *util.Minio
}

func NewGrayscaleService(grayscaleRepositories repositories.IGrayscaleManager, consulAddress string, minioClient *util.Minio) IGrayscaleModuleServive {
	return &GrayscaleModuleServive{grayscaleRepositories: grayscaleRepositories, consulAddress: consulAddress, minioClient: minioClient}
}

func (g *GrayscaleModuleServive) GetAllModulesVresions(ctx context.Context) (modulesVersionsInfo []dataModels.ModuleVersionRepsonse, err error) {
	modulesVersionsInfo, err = g.grayscaleRepositories.GetAllModuleAndVersion()
	if err != nil {
		return nil, err
	}
	return
}

// 新增模块  初始化模块可以设置： 名称， 是否使用稳定版本(初始化才会设置)，模块下面的最新versionId, 版本：pid version(不支持新增的时候直接设置成稳定版本) 默认不是稳定版本isStable： 1
func (g *GrayscaleModuleServive) AddModuleInfo(ctx context.Context, inputInfo dataModels.ModuleVersionReq) (msg string, err error) {
	moduleInfo, err := g.grayscaleRepositories.GetModuleInfoByModuleName(inputInfo.ModuleName)
	if err != nil {
		return "查询失败", err
	}
	var currentModleId int64
	if moduleInfo.Id == 0 {
		// 新增
		currentModleId, err = g.grayscaleRepositories.InsertModule(dataModels.Module{ModuleName: inputInfo.ModuleName, IsUseValid: inputInfo.IsUseValid, LatestVersionId: 0})
		if err != nil {
			return "插入module失败", err
		}
	} else {
		// 存在模板， 获取模板id
		currentModleId = moduleInfo.Id
	}

	// 校验， 针对moduleId下面的所有版本号，匹配确保版本号是最新的(不能往前设置版本号)，并且版本号符合规范
	pattern := `^\d+\.\d+\.\d+$`
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return "版本匹配内部错误", err
	}
	if !regex.MatchString(inputInfo.Version) {
		return "传入和版本号不符合规范(数字.数字.数字)", errors.New("传入和版本号不符合规范")
	}
	// 获取当前版本下面的所有version
	versionsList, err := g.grayscaleRepositories.GetAllversionUnderModule(currentModleId)
	if len(versionsList) != 0 {
		// 遍历判断
		//util.MatrixSlice
		var slice util.MatrixSlice
		for _, v := range versionsList {
			tempArr := strings.Split(v.Version, ".")
			var tempIntArr []int
			for _, value := range tempArr {
				intValue, err := strconv.Atoi(value)
				if err != nil {
					continue
				}
				tempIntArr = append(tempIntArr, intValue)
			}
			slice = append(slice, tempIntArr)
		}
		inputVersionSlice := strings.Split(inputInfo.Version, ".")
		var tempInputVersionSlice []int
		for _, v := range inputVersionSlice {
			temInt, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			tempInputVersionSlice = append(tempInputVersionSlice, temInt)
		}
		slice = append(slice, tempInputVersionSlice)
		sort.Slice(slice, slice.Less)
		var tempString [][]string
		tempString = util.IntToStr(slice)
		if inputInfo.Version != strings.Join(tempString[len(tempString)-1], ".") {
			return "设置的版本号落后，请重新提交", nil
		}
	}

	versionInfo, err := g.grayscaleRepositories.GetVersionByVersonNameAndPid(currentModleId, inputInfo.Version)
	if err != nil {
		return
	}

	if versionInfo.Id == 0 {
		// 没有版本， -> 新增版本
		newVersionId, err := g.grayscaleRepositories.InsertVersionAndUpdateModule(dataModels.Version{PId: currentModleId, Version: inputInfo.Version, IsStable: inputInfo.IsStable}, currentModleId)
		if err != nil {
			return "新增版本失败", err
		}
		return fmt.Sprintf("新增成功,moduleId:%d;  versionId:%d", currentModleId, newVersionId), nil
	} else {
		// 新增的模块版本有问题
		return "同一个模块下面，版本号重复,新增失败", errors.New("有同名模块,新增失败")
	}

	return "更新成功", nil

}

// 针对 pack application platform 设置： 稳定版本号/设置是否使用稳定版本
// 1 不需要使用稳定版本 2 需要使用稳定版本
func (g *GrayscaleModuleServive) UpdateDoModuleRequireStable(ctx context.Context, moduleId int64, isRequire int64) (msg string, err error) {
	// 判断isRequire是否符合规范: 1 否， 2 是
	constantList := []int64{1, 2}
	if !util.Contains(constantList, isRequire) {
		return "参数(1:否， 2：是),按照要求传递", nil
	}
	err = g.grayscaleRepositories.UpdateModuleIsUseValid(moduleId, isRequire)
	if err != nil {
		return "修改失败", err
	}
	return "设置成功", nil
}

// 设置模块下面的版本集合中的稳定版本
func (g *GrayscaleModuleServive) UpdateStableVersion(ctx context.Context, moduleId int64, stableVersionId int64) (msg string, err error) {
	err = g.grayscaleRepositories.UpdateIsStableInVersion(moduleId, stableVersionId)
	if err != nil {
		return err.Error(), err
	}
	return "设置成功", nil
}

// 调用grpc获取获取账号下面的模块版本
func (g *GrayscaleModuleServive) GetRemoteConfigure(ctx context.Context, moduleName string, userId int64, isServer bool) (res interface{}, err error) {
	//moduleLinkError := "xxxxx"
	// 容错，如果数据库异常，以下逻辑走不通， 返回链接重定向到错误界面
	//moduleLink = moduleLinkError
	target := fmt.Sprintf("consul://%s/%s?wait=14s", g.consulAddress, "authority-service")
	conn, err := grpc.Dial(
		//consul网络必须是通的   user_srv表示服务 wait:超时 tag是consul的tag  可以不填
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//轮询法   必须这样写   grpc在向consul发起请求时会遵循轮询法
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
	)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	userSrvClient := pb.NewUserClient(conn)
	rsp, err := userSrvClient.GetUserInfoByUserId(ctx, &pb.GetUserInfoByUserIdRequest{
		Id: userId,
	})
	fmt.Println("gprc返回的数据", rsp)
	if err != nil {
		return "", err
	}

	//  根据moduleName 找到匹配的数据模块数据
	moduleInfo, err := g.grayscaleRepositories.GetModuleInfoByModuleName(moduleName)
	if err != nil {
		return "", err
	}
	versionsInfo, err := g.grayscaleRepositories.GetAllversionUnderModule(moduleInfo.Id)
	if err != nil {
		return "", err
	}
	modulesInfoByUser := rsp.ModulesInfo
	if len(modulesInfoByUser) != 0 {
		fmt.Println("用户存在权限")
		// 当前用户存在权限
		moduleIdAndVersionIdList := strings.Split(modulesInfoByUser, "|")
		moduleIdAndVersionIdListWithList := make([][]string, len(moduleIdAndVersionIdList))
		for k, v := range moduleIdAndVersionIdList {
			moduleIdAndVersionIdListWithList[k] = strings.Split(v, "-")
		}
		// 找到指定的版本号
		var specifyVersionId string
		for _, v := range moduleIdAndVersionIdListWithList {
			// 说明用户端对于用户的版本信息录入有问题
			if len(v) != 2 {
				return
			}
			if v[0] == strconv.FormatInt(moduleInfo.Id, 10) {
				specifyVersionId = v[1]
			}
		}
		if len(specifyVersionId) != 0 {
			for _, v := range versionsInfo {
				if strconv.FormatInt(v.Id, 10) == specifyVersionId {
					// 返回指定的版本
					res, err = util.GetFromUnpkgOrFileServer(moduleName, v.Version, isServer)
					if err != nil {
						return nil, err
					}
					return res, nil
				}
			}
		} else {
			// 未找到用户端指定的权限版本，此时就当作用户没有权限
			goto NoAuthexEcute
		}
	} else {
		goto NoAuthexEcute
	}

NoAuthexEcute:
	{
		fmt.Println("用户不存在权限或者没有找到权限")
		var stableVersion string
		var latestVersion string
		for _, v := range versionsInfo {
			if v.IsStable == 2 {
				stableVersion = v.Version
			}
			if v.Id == moduleInfo.LatestVersionId {
				latestVersion = v.Version
			}
		}
		// 当前用户不存在权限， 默认采用： 1： 模块设置了使用稳定版本，使用稳定版本， 2：未设置使用稳定版本，使用最新版本
		if moduleInfo.IsUseValid == 2 {
			fmt.Println("无权限，使用了稳定版本", stableVersion)
			// 使用了稳定版本
			if len(stableVersion) != 0 {
				// 如果找到了存在稳定版本
				res, err = util.GetFromUnpkgOrFileServer(moduleName, stableVersion, isServer)
				if err != nil {
					return nil, err
				}
				return res, nil
			} else {
				// 虽然想使用稳定版本， 但是如果不存在稳定版本
				res, err = util.GetFromUnpkgOrFileServer(moduleName, latestVersion, isServer)
				if err != nil {
					return nil, err
				}
				return res, nil
			}
		} else {
			fmt.Println("无权限，使用了最新版本")
			// 使用最新版本
			if moduleInfo.LatestVersionId != 0 {
				res, err = util.GetFromUnpkgOrFileServer(moduleName, latestVersion, isServer)
				if err != nil {
					return nil, err
				}
				return res, nil
			}
		}
	}
	fmt.Print("一定执行这个打印")
	return
}

// 接受文件上传
func (g *GrayscaleModuleServive) UploadFiles(ctx context.Context, formData dataModels.FormData) (msg string, err error) {
	// 读取上传的文件
	//tmpDir, err := ioutil.TempDir("./", "form-upload")
	//if err != nil {
	//	return err.Error(), err
	//}
	//defer os.RemoveAll(tmpDir)
	//fmt.Println("assets", formData.Assets)
	//// Save the assets folder to a temporary directory
	//if err := saveFolder(formData.Assets, tmpDir); err != nil {
	//	return err.Error(), err
	//}
	// 解压文件拿到本地
	tmpFile, err := ioutil.TempFile("", "uploadAssets-*.tar.gz")
	if err != nil {
		return err.Error(), err
	}
	defer os.Remove(tmpFile.Name())
	zipFileByteArray, err := util.GetFileBytes(formData.Assets)
	if err != nil {
		return err.Error(), err
	}
	_, err = tmpFile.Write(zipFileByteArray)
	if err != nil {
		return err.Error(), err
	}
	tmpFile.Close()
	// 临时解压目录
	tmpDir, err := ioutil.TempDir("./", "uploadAssetsDir-*")
	if err != nil {
		return err.Error(), err
	}
	// 暂时先保留
	defer os.RemoveAll(tmpDir)

	// 拿到解压前的文件
	tarFile, err := os.Open(tmpFile.Name())
	if err != nil {
		return err.Error(), err
	}
	defer tarFile.Close()

	// 解压
	gzReader, err := gzip.NewReader(tarFile)
	if err != nil {
		return err.Error(), err
	}
	defer gzReader.Close()

	// 读取解压之后的文件流
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err.Error(), err
		}
		// 遍历如果是目录，新建目录
		path := filepath.Join(tmpDir, header.Name)
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return err.Error(), err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return err.Error(), err
		}
		defer file.Close()

		if _, err := io.Copy(file, tarReader); err != nil {
			return err.Error(), err
		}
	}

	// 读取 hel-meta.json 文件 获取： bucketname, minioPrefix
	helMetaPath := filepath.Join(tmpDir, "hel-meta.json")
	content, err := ioutil.ReadFile(helMetaPath)
	if err != nil {
		return err.Error(), err
	}
	// 解析JSON对象
	var metaDataJson dataModels.MetaData
	err = json.Unmarshal(content, &metaDataJson)
	if err != nil {
		return err.Error(), err
	}

	// 上传之前需要先判断数据库是否有了该版本信息(有了就默认之前文件系统中已经上传了文件)
	moduleInfo, err := g.grayscaleRepositories.GetModuleInfoByModuleName(metaDataJson.App.Name)
	if err != nil {
		return err.Error(), err
	}
	if moduleInfo.Id != 0 {
		// 存在模块
		versionInfo, err := g.grayscaleRepositories.GetVersionByVersonNameAndPid(moduleInfo.Id, metaDataJson.App.Version)
		if err != nil {
			return err.Error(), err
		}
		if versionInfo.Id != 0 {
			// 已经存在了相同的版本，不能再上传了
			return "已经有了相同的版本，请做版本升级", nil
		}
	}
	// 1：先更新数据库，因为数据库上传有更多的判断
	msg, err = g.AddModuleInfo(ctx, dataModels.ModuleVersionReq{ModuleName: metaDataJson.App.Name, IsUseValid: formData.IsUseValid, Version: metaDataJson.App.Version, IsStable: 1})
	if err != nil {
		return msg, err
	}
	// 2： 再更新静态文件服务
	err = g.minioClient.UploadFolder(ctx, metaDataJson.App.Name, tmpDir, metaDataJson.App.Version)
	if err != nil {
		fmt.Println("UploadFolder--------------------", err)
		return err.Error(), err
	}
	// 3: 也可以针对性的对本地数据做处理，再做重新上传
	return "成功上传", nil
}
