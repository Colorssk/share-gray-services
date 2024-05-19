package util

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
)

type IMatrixSlice interface {
}

type MatrixSlice [][]int

// 二位数组(int)排序
func (m MatrixSlice) Len() int {
	return len(m)
}

func (m MatrixSlice) Swap(i int, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m MatrixSlice) Less(i int, j int) bool {
	for col := 0; col < len(m[i]); col++ {
		if m[i][col] != m[j][col] {
			return m[i][col] < m[j][col]
		}
	}
	return false
}

func (m MatrixSlice) Append(i int, value []int) bool {
	m[i] = value
	return true
}

// 将 [][]string 转换为 [][]int
func StrToInt(ss [][]string) [][]int {
	ii := make([][]int, len(ss))
	for i := range ss {
		ii[i] = make([]int, len(ss[i]))
		for j := range ss[i] {
			x, err := strconv.Atoi(ss[i][j])
			if err != nil {
				// 处理转换错误
			}
			ii[i][j] = x
		}
	}
	return ii
}

// 将 [][]int 转换为 [][]string
func IntToStr(ii [][]int) [][]string {
	ss := make([][]string, len(ii))
	for i := range ii {
		ss[i] = make([]string, len(ii[i]))
		for j := range ii[i] {
			ss[i][j] = strconv.Itoa(ii[i][j])
		}
	}
	return ss
}

func Contains(arr []int64, elem int64) bool {
	for _, v := range arr {
		if v == elem {
			return true
		}
	}
	return false
}

var RedisClient *redis.Client

func InitRedis() {
	// Initialize the Redis client with appropriate configuration
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "xxx.xx.xxx.x:6379", // Replace with your Redis server address
		Password: "redispassword",     // Set password if applicable
		DB:       0,                   // Use default database
	})
}

func GetJSON(url string) (interface{}, error) {
	if RedisClient == nil {
		fmt.Println("只初始化redis一次")
		InitRedis()
	}
	// 从redis读取数据
	cachedData, err := RedisClient.Get(context.Background(), url).Result()
	if err == nil {
		fmt.Println("读取了缓存数据")
		// Data found in cache, unmarshal and return it
		var data interface{}
		if err := json.Unmarshal([]byte(cachedData), &data); err != nil {
			return nil, err
		}
		return data, nil
	} else {
		fmt.Println("读取redis数据失败", err)
	}

	// 读取minio数据
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	// 缓存数据到redis中
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	if err := RedisClient.Set(context.Background(), url, jsonData, 0).Err(); err != nil {
		return nil, err
	}

	return data, nil
}

// 针对部署在unpkg上的文件拉取
func GetFromUnpkgOrFileServer(moduleName string, version string, isServer bool) (res interface{}, err error) {
	// unpkg: http://xxx.xx.xxx.xxx:18999/pc-com-test3@1.0.1/hel_dist/hel-meta.json
	// file server: http://xxx.xx.xxx.xx:9000/pc-com-test3/1.0.0/hel-meta.json
	var uri string
	if isServer {
		// 静态文件服务
		uri = fmt.Sprintf("http://xxx.xx.xxx.xx:9000/%s/%s/hel-meta.json", moduleName, version)
	} else {
		// unpkg私服
		uri = fmt.Sprintf("http://xxx.xx.xxx.xxx:18999/%s@%s/hel_dist/hel-meta.json", moduleName, version)
	}
	res, err = GetJSON(uri)
	return
}

func GetFileBytes(file multipart.File) ([]byte, error) {
	defer file.Close()
	return ioutil.ReadAll(file)
}

func GetContentType(filename string) string {
	return mime.TypeByExtension(filepath.Ext(filename))
}
