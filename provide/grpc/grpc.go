package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	status_neko "github.com/songzhibin97/status-neko"

	"google.golang.org/grpc/metadata"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	_                status_neko.Monitor = (*Grpc)(nil)
	providerGrpcName                     = "grpc"
)

type options struct {
	dialer func(context.Context, string) (net.Conn, error)
}

type Config struct {
	URL              string            `json:"url"`
	TLS              bool              `json:"tls"`
	ProtoServiceName string            `json:"proto_service_name"`
	ProtoMethod      string            `json:"proto_method"`
	ProtoContents    map[string]string `json:"proto_content"`
	Request          string            `json:"request"`
	Metadata         map[string]string `json:"metadata"`
}

type Grpc struct {
	config Config

	options *options
}

func SetDialer(dialer func(context.Context, string) (net.Conn, error)) status_neko.Option[*options] {
	return func(o *options) {
		o.dialer = dialer
	}
}

func NewGrpc(config Config, opts ...status_neko.Option[*options]) *Grpc {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return &Grpc{
		config:  config,
		options: o,
	}
}

func (g Grpc) Name() string {
	return providerGrpcName
}

func (g Grpc) Check(ctx context.Context) (interface{}, error) {
	// 1. Parse the ProtoContent directly without writing to a file
	parser := protoparse.Parser{
		Accessor: protoparse.FileContentsFromMap(g.config.ProtoContents),
	}
	fileNames := make([]string, 0, len(g.config.ProtoContents))
	for fileName := range g.config.ProtoContents {
		fileNames = append(fileNames, fileName)

	}
	fds, err := parser.ParseFiles(fileNames...)
	if err != nil {
		return nil, fmt.Errorf("解析 proto 内容失败: %w", err)
	}

	// 2. Find the service description
	var service *desc.ServiceDescriptor
	for _, fd := range fds {
		service = fd.FindService(g.config.ProtoServiceName)
		if service != nil {
			break
		}
	}
	if service == nil {
		return nil, fmt.Errorf("在 proto 中未找到服务 %s", g.config.ProtoServiceName)
	}

	// 3. Find the method
	method := service.FindMethodByName(g.config.ProtoMethod)
	if method == nil {
		return nil, fmt.Errorf("在服务 %s 中未找到方法 %s", g.config.ProtoServiceName, g.config.ProtoMethod)
	}

	var opts []grpc.DialOption
	if g.config.TLS {
		creds := credentials.NewClientTLSFromCert(nil, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add custom dialer for testing
	if g.options.dialer != nil {
		opts = append(opts, grpc.WithContextDialer(g.options.dialer))
	}

	conn, err := grpc.DialContext(ctx, g.config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("连接 gRPC 服务器失败: %w", err)
	}
	defer conn.Close()

	// 5. Create dynamic client
	stub := grpcdynamic.NewStub(conn)

	// 6. Construct request
	reqMsg := dynamic.NewMessage(method.GetInputType())
	err = json.Unmarshal([]byte(g.config.Request), reqMsg)
	if err != nil {
		return nil, fmt.Errorf("解析请求 JSON 失败: %w", err)
	}

	// 7. Execute gRPC call
	start := time.Now()

	respMsg, err := stub.InvokeRpc(metadata.NewOutgoingContext(ctx, metadata.New(g.config.Metadata)), method, reqMsg)
	if err != nil {
		return nil, fmt.Errorf("gRPC 调用失败: %w", err)
	}

	// Convert response to JSON
	responseJSON, err := respMsg.(*dynamic.Message).MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("响应序列化失败: %w", err)
	}

	var jsonObj interface{}
	err = json.Unmarshal(responseJSON, &jsonObj)
	if err != nil {
		return nil, fmt.Errorf("响应不是有效的 JSON: %w", err)
	}

	result := map[string]interface{}{
		"url":           g.config.URL,
		"method":        g.config.ProtoMethod,
		"response":      jsonObj,
		"response_time": time.Since(start),
	}

	return result, nil
}
