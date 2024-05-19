package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"grayscaleService/common"
	"grayscaleService/dataModels"
	"strconv"
)

type IGrayscaleManager interface {
	Conn() error
	// 获取模块一级信息
	// 获取版本信息
	GetAllModuleAndVersion() ([]dataModels.ModuleVersionRepsonse, error)
	// 单个module信息
	GetModuleInfoByModuleName(string) (dataModels.Module, error)
	// 单个版本信息
	GetVersionByVersonNameAndPid(int64, string) (dataModels.Version, error)
	// 单个模块插入
	InsertModule(dataModels.Module) (int64, error)
	// 单个版本插入
	InsertVersionAndUpdateModule(dataModels.Version, int64) (int64, error)
	// 获取模块下面的所有版本信息
	GetAllversionUnderModule(int64) ([]*dataModels.Version, error)
	// 更新 module 是否需要稳定版本字段
	UpdateModuleIsUseValid(int64, int64) error
	// 设置模块下面最新版本
	UpdateIsStableInVersion(int64, int64) error
}

type GrayscaleManager struct {
	moduleTable  string
	versionTable string
	mysqlConn    *sql.DB
}

func NewGrayscaleMansger(moduleTable string, versionTable string, db *sql.DB) IGrayscaleManager {
	return &GrayscaleManager{moduleTable: moduleTable, versionTable: versionTable, mysqlConn: db}
}
func (g *GrayscaleManager) Conn() (err error) {
	if g.mysqlConn == nil {
		mysql, err := common.NewMysqlConn("managePlatform")
		if err != nil {
			return err
		}
		g.mysqlConn = mysql
	}
	if g.moduleTable == "" || g.versionTable == "" {
		g.moduleTable = "module"
		g.versionTable = "version"
	}
	return
}

// SELECT module.id AS moduleId, module.moduleName, module.isUseValid, module.latestVersionId ,version.id AS versionId, version.version,version.isStable
// FROM module as module
// JOIN version as version ON module.id = version.pid;
// get module and version
func (g *GrayscaleManager) GetAllModuleAndVersion() (moduleVersionList []dataModels.ModuleVersionRepsonse, err error) {
	if err = g.Conn(); err != nil {
		return
	}
	sql := fmt.Sprintf("SELECT module.id AS moduleId, module.moduleName, module.isUseValid, module.latestVersionId , version.id AS versionId, version.version,version.isStable FROM %s as module JOIN %s as version ON module.id = version.pid ORDER BY module.id ASC;", g.moduleTable, g.versionTable)
	fmt.Println(sql)
	row, err := g.mysqlConn.Query(sql)
	defer row.Close()
	if err != nil {
		return
	}
	results := common.GetResultRows(row)
	if len(results) == 0 {
		return
	}
	sqlFlatList := make([]*dataModels.ModuleVersionSql, len(results))
	for k, v := range results {
		singleModuleVersion := &dataModels.ModuleVersionSql{}
		common.DataToStructByTagSql(v, singleModuleVersion)
		sqlFlatList[k] = singleModuleVersion
	}
	// 对于相同的moduleId, group 成一条
	resultWithModuleId := make(map[int64]*dataModels.ModuleVersionRepsonse)
	for _, v := range sqlFlatList {
		versionResp := &dataModels.VersionResponse{VersionId: v.VersionId, Version: v.Version, IsStable: v.IsStable}
		var versionRespList []*dataModels.VersionResponse
		versionRespList = append(versionRespList, versionResp)
		if _, ok := resultWithModuleId[v.ModuleId]; ok {
			resultWithModuleId[v.ModuleId].VersionList = append(resultWithModuleId[v.ModuleId].VersionList, versionResp)
		} else {
			resultWithModuleId[v.ModuleId] = &dataModels.ModuleVersionRepsonse{ModuleId: v.ModuleId, ModuleName: v.ModuleName, IsUseValid: v.IsUseValid, LatestVersionId: v.LatestVersionId, VersionList: versionRespList}
		}
	}
	for _, v := range resultWithModuleId {
		moduleVersionList = append(moduleVersionList, *v)
	}
	return
}

// 查询module
func (g *GrayscaleManager) GetModuleInfoByModuleName(moduleName string) (moduleInfo dataModels.Module, err error) {
	if err = g.Conn(); err != nil {
		return
	}
	sql := fmt.Sprintf("SELECT * FROM %s WHERE moduleName=\"%s\";", g.moduleTable, moduleName)
	row, err := g.mysqlConn.Query(sql)
	defer row.Close()
	if err != nil {
		return
	}
	results := common.GetResultRows(row)
	fmt.Println(results)
	if len(results) == 0 {
		return dataModels.Module{}, nil
	}
	moduleTemp := &dataModels.Module{}
	common.DataToStructByTagSql(results[0], moduleTemp)
	moduleInfo = *moduleTemp
	return
}

// 查询version
func (g *GrayscaleManager) GetVersionByVersonNameAndPid(versionPId int64, versionName string) (versionInfo dataModels.Version, err error) {
	if err = g.Conn(); err != nil {
		return
	}
	sql := fmt.Sprintf("SELECT * FROM %s WHERE pid=%s AND version=\"%s\";", g.versionTable, strconv.FormatInt(versionPId, 10), versionName)
	row, err := g.mysqlConn.Query(sql)
	defer row.Close()
	if err != nil {
		return
	}
	results := common.GetResultRows(row)
	if len(results) == 0 {
		return dataModels.Version{}, nil
	}
	versionTemp := &dataModels.Version{}
	common.DataToStructByTagSql(results[0], versionTemp)
	versionInfo = *versionTemp
	return
}

// 新增module
// INSERT INTO module(moduleName, isUseValid, latestVersionId)
// (SELECT "testInsert", 1, 1);
func (g *GrayscaleManager) InsertModule(moduleInfo dataModels.Module) (newModuleId int64, err error) {
	if err = g.Conn(); err != nil {
		return
	}
	sql := fmt.Sprintf("INSERT INTO %s(moduleName, isUseValid, latestVersionId) (SELECT \"%s\", %d, %d);", g.moduleTable, moduleInfo.ModuleName, moduleInfo.IsUseValid, moduleInfo.LatestVersionId)
	fmt.Println(sql)
	stmt, err := g.mysqlConn.Prepare(sql)
	if err != nil {
		return
	}
	result, err := stmt.Exec()
	if err != nil {
		return
	}
	return result.LastInsertId()
}

// 新增version  + 事务, 保证原子性: 对新增的version 插入module，最为latestVerionId
func (g *GrayscaleManager) InsertVersionAndUpdateModule(versionInfo dataModels.Version, moduleId int64) (newVersionId int64, err error) {
	if err = g.Conn(); err != nil {
		return
	}
	sql := fmt.Sprintf("INSERT INTO %s(pid, version, isStable) (SELECT %d, \"%s\", %d);", g.versionTable, versionInfo.PId, versionInfo.Version, 1)
	tx, err := g.mysqlConn.Begin()
	if err != nil {
		return 0, err
	}
	versionInsertResult, err := tx.Exec(sql)
	if err != nil {
		tx.Rollback()
		return
	}
	latestVersionId, err := versionInsertResult.LastInsertId()
	sql = fmt.Sprintf("UPDATE %s SET latestVersionId=%d WHERE id=%d;", g.moduleTable, latestVersionId, moduleId)
	fmt.Println(sql)
	updateModuleResult, err := tx.Exec(sql)
	if err != nil {
		tx.Rollback()
		return
	}

	updateIndex, err := updateModuleResult.RowsAffected()
	if err != nil || updateIndex == 0 {
		tx.Rollback()
		return 0, errors.New("新增失败")
	}

	err = tx.Commit()
	if err != nil {
		return 0, errors.New("新增失败")
	}
	return latestVersionId, nil
}

// 获取模块id下面的所有version
func (g *GrayscaleManager) GetAllversionUnderModule(moduleId int64) (versions []*dataModels.Version, err error) {
	if err = g.Conn(); err != nil {
		return
	}
	sql := fmt.Sprintf("SELECT * FROM %s WHERE pid=%d", g.versionTable, moduleId)
	row, err := g.mysqlConn.Query(sql)
	defer row.Close()
	if err != nil {
		return
	}
	results := common.GetResultRows(row)
	if len(results) == 0 {
		return nil, nil
	}
	for _, v := range results {
		versionInfo := &dataModels.Version{}
		common.DataToStructByTagSql(v, versionInfo)
		versions = append(versions, versionInfo)
	}
	return

}

// 更新module表中isUseValid字段
func (g *GrayscaleManager) UpdateModuleIsUseValid(moduleId int64, isUseValidReq int64) (err error) {
	if err = g.Conn(); err != nil {
		return errors.New("更新失败")
	}
	tx, err := g.mysqlConn.Begin()
	if err != nil {
		return errors.New("更新失败")
	}
	sql := fmt.Sprintf("UPDATE %s SET isUseValid=%d WHERE id=%d", g.moduleTable, isUseValidReq, moduleId)
	updateResult, err := tx.Exec(sql)
	if err != nil {
		return errors.New("更新失败")
	}
	affectIndex, err := updateResult.RowsAffected()
	if err != nil || affectIndex == 0 {
		return errors.New("更新失败")
	}
	err = tx.Commit()
	if err != nil {
		return errors.New("更新失败")
	}
	return
}

// 如果存在，第一条生效，重置之前的稳定版本()，之后执行第二条， 设置需要设置成稳定版本的版本
// UPDATE version SET isStable = 1 WHERE pid=9 AND isStable = 2;
// UPDATE version SET isStable = 2 WHERE id=16;
func (g *GrayscaleManager) UpdateIsStableInVersion(moduleId int64, versionId int64) (err error) {
	if err = g.Conn(); err != nil {
		return
	}
	tx, err := g.mysqlConn.Begin()
	if err != nil {
		return errors.New("设置失败")
	}
	checkExistSql := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE pid=%d AND isStable=2", g.versionTable, moduleId)
	row := tx.QueryRow(checkExistSql)

	var count int
	err = row.Scan(&count)
	if err != nil {
		return errors.New("设置失败")
	}
	if count > 0 {
		// 存在需要更新
		sqlCancel := fmt.Sprintf("UPDATE %s SET isStable=1 WHERE pid=%d AND isStable=2;", g.versionTable, moduleId)

		// 重置 执行错误 回退
		cancelResult, err := tx.Exec(sqlCancel)
		if err != nil {
			tx.Rollback()
			return errors.New("设置失败")
		}
		cancelIndex, err := cancelResult.RowsAffected()
		if err != nil || cancelIndex == 0 {
			tx.Rollback()
			return errors.New("设置失败")
		}
	}

	sqlReSet := fmt.Sprintf("UPDATE %s SET isStable = 2 WHERE id=%d AND pid=%d;", g.versionTable, versionId, moduleId)
	resultSetResult, err := tx.Exec(sqlReSet)
	if err != nil {
		fmt.Println(err, 4)
		tx.Rollback()
		return errors.New("设置失败")
	}
	// 未设置到指定的版本号，重置取消操作
	updateIndex, err := resultSetResult.RowsAffected()
	if err != nil || updateIndex == 0 {
		tx.Rollback()
		return errors.New("指定的versionID未检索到")
	}
	err = tx.Commit()
	if err != nil {
		return errors.New("设置失败")
	}
	return
}

// 根据moduleName
