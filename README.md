# status-neko

服务状态检测工具,用于拨测检测服务是否正常运行

支持的检测方式:
- HTTP
- TCP
- ICMP
- DNS
- GRPC
- CertificateExpires
- Kafka
- Redis
- Mysql
- Mongodb
- MSSQL
- PGSQL
- MQTT

## 使用

所有支持项都实现了 `Monitor` 接口,可以通过实现该接口来扩展新的检测项

```go
type Monitor interface {
	// Name returns the name of the monitor.
	Name() string

	// Check returns the status of the service.
	Check(ctx context.Context) (interface{}, error)
}
```

例如用于检测 HTTP 服务是否正常运行

```go
package main

import (
	"context"
	"fmt"
	"github.com/songzhibin97/status-neko/provide/http"
)

func main() {
	client := http.NewHTTP(http.Config{
		URL:    "http://baidu.com",
		Method: http.GET,
	})

	result, err := client.Check(context.Background())
	if err != nil {
		panic(err)
	}

	/*
	    <html>
		<meta http-equiv="refresh" content="0;url=http://www.baidu.com/">
		</html>
	*/
	fmt.Println(result)
}

```