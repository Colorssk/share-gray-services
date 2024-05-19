package util

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"net/http"
)

type ICosuleClient interface {
	CallService(string, string, string) (string, error)
}

// ConsulClient 封装的Consul客户端
type ConsulClient struct {
	client *api.Client
}

// NewConsulClient 创建Consul客户端实例
func NewConsulClient(consulAddr string) (ICosuleClient, error) {
	// 创建连接Consul的配置
	config := api.DefaultConfig()
	config.Address = consulAddr

	// 创建Consul客户端实例
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &ConsulClient{client: client}, nil
}

// GetServiceAddress 获取服务实例的地址
func (c *ConsulClient) GetServiceAddress(serviceName string) (string, error) {
	// 通过服务名称查找服务实例
	entries, _, err := c.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return "", err
	}

	// 随机选择一个服务实例并返回其地址
	if len(entries) > 0 {
		return fmt.Sprintf("%s:%d", entries[0].Service.Address, entries[0].Service.Port), nil
	}

	return "", fmt.Errorf("service '%s' not found", serviceName)
}

// CallService 调用服务
func (c *ConsulClient) CallService(serviceName, endpoint, method string) (string, error) {
	// 获取服务实例地址
	address, err := c.GetServiceAddress(serviceName)
	if err != nil {
		return "", err
	}

	// 创建HTTP请求
	url := fmt.Sprintf("http://%s/%s", address, endpoint)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}

	// 发送HTTP请求
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
