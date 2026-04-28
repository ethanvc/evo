package dkit

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ethanvc/evo/xobs"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func AddGrpcCmd(rootCmd *cobra.Command) {
	// can use https://grpcb.in/ to test this command
	cmd := &cobra.Command{
		Use:   "grpc",
		Short: "gRPC command-line client",
		Long: `grpc is a command-line gRPC client that supports:

  Send a request:
    Resolve proto types via server reflection, serialize a JSON body, invoke the method,
    and print the response as JSON. Use --sub-type to bypass reflection with a raw codec.

  Reflection queries (--query):
    list-svr    List all services exposed by the server
    list-method List all methods of a service (requires --svr)
    show-method Show the request/response schema of a method (requires --method)`,
	}
	host := cmd.Flags().String("host", "127.0.0.1:8888", "server address in host:port format")
	method := cmd.Flags().String("method", "", "method path, e.g. /package.Service/Method")
	body := cmd.Flags().String("body", "", "request body as JSON; prefix with @ to read from a file (e.g. @req.json)")
	subType := cmd.Flags().String("sub-type", "", "raw codec name (e.g. proto, json); omit to auto-resolve via reflection")
	query := cmd.Flags().String("query", "", "reflection query: list-svr | list-method | show-method")
	svr := cmd.Flags().String("svr", "", "service name in package.Service format (required for list-method)")
	tls := cmd.Flags().Bool("tls", false, "enable TLS; by default the connection is plaintext")
	initConnWin := cmd.Flags().Int32("initial-conn-window-size", 0, "HTTP/2 initial connection flow-control window (bytes); 0 uses gRPC default")
	initWin := cmd.Flags().Int32("initial-window-size", 0, "HTTP/2 initial stream flow-control window (bytes); 0 uses gRPC default")
	count := cmd.Flags().Int("count", 1, "number of times to send the request")
	headers := cmd.Flags().StringArrayP("header", "H", nil, "gRPC metadata header in 'key: value' format (repeatable); use @file to read from a file")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return GrpcMain(&GrpcMainReq{
			Host:                  *host,
			Body:                  *body,
			Method:                *method,
			SubType:               *subType,
			Query:                 *query,
			Svr:                   *svr,
			TLS:                   *tls,
			InitialConnWindowSize: *initConnWin,
			InitialWindowSize:     *initWin,
			Count:                 *count,
			Headers:               *headers,
		})
	}
	rootCmd.AddCommand(cmd)
}

// RawCodec is a gRPC codec that passes bytes through without marshaling.
type RawCodec struct {
	name string
}

func NewRawCodec(name string) *RawCodec {
	return &RawCodec{name: name}
}

func (c *RawCodec) Marshal(v any) ([]byte, error) {
	switch val := v.(type) {
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	default:
		return nil, fmt.Errorf("raw codec: unsupported marshal type %T", v)
	}
}

func (c *RawCodec) Unmarshal(data []byte, v any) error {
	switch ptr := v.(type) {
	case *[]byte:
		*ptr = data
		return nil
	default:
		return fmt.Errorf("raw codec: expect *[]byte, received %T", v)
	}
}

func (c *RawCodec) Name() string {
	return c.name
}

type GrpcMainReq struct {
	Host                  string
	Body                  string
	Method                string
	SubType               string
	Query                 string
	Svr                   string
	TLS                   bool
	InitialConnWindowSize int32    // 0: omit WithInitialConnWindowSize
	InitialWindowSize     int32    // 0: omit WithInitialWindowSize
	Count                 int      // number of times to send the request; default 1
	Headers               []string // "key: value" pairs sent as gRPC metadata
}

var validQueryValues = map[string]bool{
	"list-svr":    true,
	"list-method": true,
	"show-method": true,
}

func (r *GrpcMainReq) Validate() error {
	if err := validateHost(r.Host); err != nil {
		return err
	}
	if r.Query != "" {
		return r.validateQueryMode()
	}
	return r.validateSendMode()
}

func validateHost(host string) error {
	if strings.TrimSpace(host) == "" {
		return xobs.New(codes.InvalidArgument, "MissingHost").SetMsg("--host must not be empty")
	}
	if !strings.Contains(host, ":") {
		return xobs.New(codes.InvalidArgument, "InvalidHost").SetMsg("--host must be in host:port format")
	}
	return nil
}

func (r *GrpcMainReq) validateQueryMode() error {
	if !validQueryValues[r.Query] {
		return xobs.New(codes.InvalidArgument, "InvalidQueryValue").
			SetMsg(fmt.Sprintf("--query %q is invalid; allowed values: list-svr | list-method | show-method", r.Query))
	}
	if r.Query == "list-method" && strings.TrimSpace(r.Svr) == "" {
		return xobs.New(codes.InvalidArgument, "MissingSvr").
			SetMsg("--svr is required when --query=list-method")
	}
	if r.Query == "show-method" && strings.TrimSpace(r.Method) == "" {
		return xobs.New(codes.InvalidArgument, "MissingMethod").
			SetMsg("--method is required when --query=show-method")
	}
	return nil
}

func (r *GrpcMainReq) validateSendMode() error {
	if strings.TrimSpace(r.Method) == "" {
		return xobs.New(codes.InvalidArgument, "MissingMethod").
			SetMsg("--method must not be empty; expected format: /package.Service/Method")
	}
	if _, _, err := parseMethodPath(r.Method); err != nil {
		return xobs.New(codes.InvalidArgument, "InvalidMethod").SetMsg(err.Error())
	}
	return nil
}

func resolveBody(body string) ([]byte, error) {
	if strings.HasPrefix(body, "@") {
		return os.ReadFile(body[1:])
	}
	return []byte(body), nil
}

func GrpcMain(req *GrpcMainReq) error {
	if err := req.Validate(); err != nil {
		return err
	}
	if req.Query != "" {
		return queryByReflect(req)
	}
	return sendRequest(req)
}

func queryByReflect(req *GrpcMainReq) error {
	switch req.Query {
	case "list-svr":
		return querySvrList(req)
	case "list-method":
		return queryMethodList(req)
	case "show-method":
		return queryShowMethod(req)
	default:
		// unreachable: already rejected by Validate()
		return xobs.New(codes.InvalidArgument, "InvalidQueryValue").SetMsg("invalid query value")
	}
}

func queryMethodList(req *GrpcMainReq) error {
	ctx := context.Background()
	rc, err := NewReflectionClient(ctx, newGrpcClientConfig(req))
	if err != nil {
		return err
	}
	defer rc.Close()
	methods, err := rc.ListMethods(ctx, req.Svr)
	if err != nil {
		return err
	}
	for _, m := range methods {
		fmt.Println(m)
	}
	return nil
}

type ReflectionClient interface {
	ListServices(ctx context.Context) ([]string, error)
	ListMethods(ctx context.Context, service string) ([]string, error)
	GetFileDescriptorsBySymbol(ctx context.Context, symbol string) ([]*descriptorpb.FileDescriptorProto, error)
	Close() error
}

// NewReflectionClient tries v1 and v1alpha with independent connections,
// returns the first that succeeds, or both errors if both fail.
func NewReflectionClient(ctx context.Context, conf *GrpcClientConfig) (ReflectionClient, error) {
	v1, err1 := newReflectionClientV1(conf)
	if err1 == nil {
		_, err1 = v1.ListServices(ctx)
		if err1 == nil {
			return v1, nil
		}
		v1.Close()
	}

	v1alpha, err2 := newReflectionClientV1Alpha(conf)
	if err2 == nil {
		_, err2 = v1alpha.ListServices(ctx)
		if err2 == nil {
			return v1alpha, nil
		}
		v1alpha.Close()
	}

	return nil, fmt.Errorf("v1: %w; v1alpha: %w", err1, err2)
}

type reflectionClientV1 struct {
	cc *grpc.ClientConn
}

func newReflectionClientV1(conf *GrpcClientConfig) (*reflectionClientV1, error) {
	cc, err := NewGrpcClient(conf)
	if err != nil {
		return nil, err
	}
	return &reflectionClientV1{cc: cc}, nil
}

func (c *reflectionClientV1) Close() error { return c.cc.Close() }

func (c *reflectionClientV1) ListServices(ctx context.Context) ([]string, error) {
	stream, err := reflectionv1.NewServerReflectionClient(c.cc).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_ListServices{},
	}); err != nil {
		return nil, err
	}
	stream.CloseSend()
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	list := resp.GetListServicesResponse()
	if list == nil {
		return nil, fmt.Errorf("unexpected response: %v", resp.GetMessageResponse())
	}
	var names []string
	for _, svc := range list.GetService() {
		names = append(names, svc.GetName())
	}
	return names, nil
}

func (c *reflectionClientV1) GetFileDescriptorsBySymbol(ctx context.Context, symbol string) ([]*descriptorpb.FileDescriptorProto, error) {
	stream, err := reflectionv1.NewServerReflectionClient(c.cc).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}); err != nil {
		return nil, err
	}
	stream.CloseSend()
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		return nil, fmt.Errorf("unexpected response: %v", resp.GetMessageResponse())
	}
	return parseFileDescriptors(fdResp.GetFileDescriptorProto())
}

func (c *reflectionClientV1) ListMethods(ctx context.Context, service string) ([]string, error) {
	fds, err := c.GetFileDescriptorsBySymbol(ctx, service)
	if err != nil {
		return nil, err
	}
	return extractMethods(fds, service)
}

type reflectionClientV1Alpha struct {
	cc *grpc.ClientConn
}

func newReflectionClientV1Alpha(conf *GrpcClientConfig) (*reflectionClientV1Alpha, error) {
	cc, err := NewGrpcClient(conf)
	if err != nil {
		return nil, err
	}
	return &reflectionClientV1Alpha{cc: cc}, nil
}

func (c *reflectionClientV1Alpha) Close() error { return c.cc.Close() }

func (c *reflectionClientV1Alpha) ListServices(ctx context.Context) ([]string, error) {
	stream, err := reflectionv1alpha.NewServerReflectionClient(c.cc).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, xobs.New(codes.Internal, "CallServerReflectionInfoErr").SetMsg(err.Error())
	}
	if err = stream.Send(&reflectionv1alpha.ServerReflectionRequest{
		MessageRequest: &reflectionv1alpha.ServerReflectionRequest_ListServices{},
	}); err != nil {
		return nil, xobs.New(codes.Internal, "CallStreamSendErr").SetMsg(err.Error())
	}
	stream.CloseSend()
	resp, err := stream.Recv()
	if err != nil {
		return nil, xobs.New(codes.Internal, "CallStreamRecvErr").SetMsg(err.Error())
	}
	list := resp.GetListServicesResponse()
	if list == nil {
		return nil, fmt.Errorf("unexpected response: %v", resp.GetMessageResponse())
	}
	var names []string
	for _, svc := range list.GetService() {
		names = append(names, svc.GetName())
	}
	return names, nil
}

func (c *reflectionClientV1Alpha) GetFileDescriptorsBySymbol(ctx context.Context, symbol string) ([]*descriptorpb.FileDescriptorProto, error) {
	stream, err := reflectionv1alpha.NewServerReflectionClient(c.cc).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&reflectionv1alpha.ServerReflectionRequest{
		MessageRequest: &reflectionv1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}); err != nil {
		return nil, err
	}
	stream.CloseSend()
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		return nil, fmt.Errorf("unexpected response: %v", resp.GetMessageResponse())
	}
	return parseFileDescriptors(fdResp.GetFileDescriptorProto())
}

func (c *reflectionClientV1Alpha) ListMethods(ctx context.Context, service string) ([]string, error) {
	fds, err := c.GetFileDescriptorsBySymbol(ctx, service)
	if err != nil {
		return nil, err
	}
	return extractMethods(fds, service)
}

func parseFileDescriptors(rawDescs [][]byte) ([]*descriptorpb.FileDescriptorProto, error) {
	var fds []*descriptorpb.FileDescriptorProto
	for _, raw := range rawDescs {
		fd := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(raw, fd); err != nil {
			return nil, fmt.Errorf("unmarshal file descriptor: %w", err)
		}
		fds = append(fds, fd)
	}
	return fds, nil
}

func extractMethods(fds []*descriptorpb.FileDescriptorProto, service string) ([]string, error) {
	for _, fd := range fds {
		for _, svc := range fd.GetService() {
			fqn := fd.GetPackage() + "." + svc.GetName()
			if fqn != service {
				continue
			}
			var methods []string
			for _, m := range svc.GetMethod() {
				methods = append(methods, fqn+"/"+m.GetName())
			}
			return methods, nil
		}
	}
	return nil, fmt.Errorf("service %q not found in file descriptors", service)
}

func queryShowMethod(req *GrpcMainReq) error {
	ctx := context.Background()
	rc, err := NewReflectionClient(ctx, newGrpcClientConfig(req))
	if err != nil {
		return err
	}
	defer rc.Close()

	svcName, methodName, err := parseMethodPath(req.Method)
	if err != nil {
		return err
	}

	fds, err := rc.GetFileDescriptorsBySymbol(ctx, svcName)
	if err != nil {
		return err
	}

	md, err := findMethodDescriptor(fds, svcName, methodName)
	if err != nil {
		return err
	}

	inputFQ := md.GetInputType()
	outputFQ := md.GetOutputType()
	fmt.Printf("rpc %s(%s%s) returns (%s%s)\n\n",
		md.GetName(),
		streamPrefix(md.GetClientStreaming()),
		shortTypeName(inputFQ),
		streamPrefix(md.GetServerStreaming()),
		shortTypeName(outputFQ),
	)

	resolver := &messageResolver{
		rc:      rc,
		ctx:     ctx,
		fdMap:   make(map[string]*descriptorpb.FileDescriptorProto),
		printed: make(map[string]bool),
	}
	for _, fd := range fds {
		resolver.fdMap[fd.GetName()] = fd
	}
	resolver.resolveAndPrint(inputFQ)
	resolver.resolveAndPrint(outputFQ)
	return nil
}

type messageResolver struct {
	rc      ReflectionClient
	ctx     context.Context
	fdMap   map[string]*descriptorpb.FileDescriptorProto
	printed map[string]bool
}

func (r *messageResolver) resolveAndPrint(typeFQ string) {
	normalized := strings.TrimPrefix(typeFQ, ".")
	if r.printed[normalized] {
		return
	}
	r.printed[normalized] = true

	msg := findMessageInFdMap(r.fdMap, typeFQ)
	if msg == nil {
		moreFds, err := r.rc.GetFileDescriptorsBySymbol(r.ctx, normalized)
		if err == nil {
			for _, fd := range moreFds {
				r.fdMap[fd.GetName()] = fd
			}
			msg = findMessageInFdMap(r.fdMap, typeFQ)
		}
	}
	if msg == nil {
		return
	}

	printMessageDescriptor(msg, typeFQ)

	for _, f := range msg.GetField() {
		if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			r.resolveAndPrint(f.GetTypeName())
		}
	}
}

// normalizeMethodPath ensures the path starts with "/" as required by HTTP/2 :path.
func normalizeMethodPath(method string) string {
	if !strings.HasPrefix(method, "/") {
		return "/" + method
	}
	return method
}

func parseMethodPath(method string) (service, methodName string, err error) {
	method = strings.TrimPrefix(method, "/")
	idx := strings.LastIndex(method, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid method path %q, expected format: package.Service/Method", method)
	}
	return method[:idx], method[idx+1:], nil
}

func findMethodDescriptor(fds []*descriptorpb.FileDescriptorProto, service, method string) (*descriptorpb.MethodDescriptorProto, error) {
	for _, fd := range fds {
		for _, svc := range fd.GetService() {
			fqn := fd.GetPackage() + "." + svc.GetName()
			if fqn != service {
				continue
			}
			for _, m := range svc.GetMethod() {
				if m.GetName() == method {
					return m, nil
				}
			}
			return nil, fmt.Errorf("method %q not found in service %q", method, service)
		}
	}
	return nil, fmt.Errorf("service %q not found in file descriptors", service)
}

func findMessageInFdMap(fdMap map[string]*descriptorpb.FileDescriptorProto, fqn string) *descriptorpb.DescriptorProto {
	fqn = strings.TrimPrefix(fqn, ".")
	for _, fd := range fdMap {
		pkg := fd.GetPackage()
		for _, msg := range fd.GetMessageType() {
			if found := findNestedMessage(msg, pkg+"."+msg.GetName(), fqn); found != nil {
				return found
			}
		}
	}
	return nil
}

func findNestedMessage(msg *descriptorpb.DescriptorProto, prefix, target string) *descriptorpb.DescriptorProto {
	if prefix == target {
		return msg
	}
	for _, nested := range msg.GetNestedType() {
		if found := findNestedMessage(nested, prefix+"."+nested.GetName(), target); found != nil {
			return found
		}
	}
	return nil
}

func printMessageDescriptor(msg *descriptorpb.DescriptorProto, fqn string) {
	fmt.Printf("message %s {\n", shortTypeName(fqn))
	for _, f := range msg.GetField() {
		label := ""
		if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
			label = "repeated "
		}
		fmt.Printf("  %s%s %s = %d;\n", label, protoFieldTypeName(f), f.GetName(), f.GetNumber())
	}
	fmt.Println("}")
	fmt.Println()
}

func shortTypeName(fqn string) string {
	fqn = strings.TrimPrefix(fqn, ".")
	if idx := strings.LastIndex(fqn, "."); idx >= 0 {
		return fqn[idx+1:]
	}
	return fqn
}

func streamPrefix(streaming bool) string {
	if streaming {
		return "stream "
	}
	return ""
}

func protoFieldTypeName(f *descriptorpb.FieldDescriptorProto) string {
	switch f.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		return shortTypeName(f.GetTypeName())
	default:
		return strings.TrimPrefix(strings.ToLower(f.GetType().String()), "type_")
	}
}

func querySvrList(req *GrpcMainReq) error {
	ctx := context.Background()
	rc, err := NewReflectionClient(ctx, newGrpcClientConfig(req))
	if err != nil {
		return err
	}
	defer rc.Close()
	names, err := rc.ListServices(ctx)
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

// resolveMethodMessages uses server reflection to look up the input and output proto message
// types for the given method, returning a zero-value input message and the output descriptor.
func resolveMethodMessages(ctx context.Context, conf *GrpcClientConfig, method string) (inputMsg proto.Message, outputMsgDesc protoreflect.MessageDescriptor, err error) {
	svcName, methodName, err := parseMethodPath(method)
	if err != nil {
		return nil, nil, err
	}
	rc, err := NewReflectionClient(ctx, conf)
	if err != nil {
		return nil, nil, err
	}
	defer rc.Close()

	fds, err := rc.GetFileDescriptorsBySymbol(ctx, svcName)
	if err != nil {
		return nil, nil, err
	}
	md, err := findMethodDescriptor(fds, svcName, methodName)
	if err != nil {
		return nil, nil, err
	}
	registry, err := buildDescriptorRegistry(fds)
	if err != nil {
		return nil, nil, err
	}

	inputTypeName := protoreflect.FullName(strings.TrimPrefix(md.GetInputType(), "."))
	outputTypeName := protoreflect.FullName(strings.TrimPrefix(md.GetOutputType(), "."))

	inputDesc, err := registry.FindDescriptorByName(inputTypeName)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve input type %s: %w", inputTypeName, err)
	}
	outputDesc, err := registry.FindDescriptorByName(outputTypeName)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve output type %s: %w", outputTypeName, err)
	}

	inputMsgDesc, ok := inputDesc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, nil, fmt.Errorf("input type %s is not a message", inputTypeName)
	}
	outputMsgDescVal, ok := outputDesc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, nil, fmt.Errorf("output type %s is not a message", outputTypeName)
	}

	return dynamicpb.NewMessage(inputMsgDesc), outputMsgDescVal, nil
}

func parseHeaders(raw []string) (metadata.MD, error) {
	md := metadata.MD{}
	for _, h := range raw {
		if strings.HasPrefix(h, "@") {
			if err := loadHeadersFromFile(h[1:], &md); err != nil {
				return nil, err
			}
			continue
		}
		if err := addHeader(&md, h); err != nil {
			return nil, err
		}
	}
	return md, nil
}

func loadHeadersFromFile(path string, md *metadata.MD) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read header file %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := addHeader(md, line); err != nil {
			return fmt.Errorf("header file %s: %w", path, err)
		}
	}
	return nil
}

func addHeader(md *metadata.MD, h string) error {
	k, v, ok := strings.Cut(h, ":")
	if !ok {
		return fmt.Errorf("invalid header %q: expected 'key: value'", h)
	}
	md.Append(strings.TrimSpace(k), strings.TrimSpace(v))
	return nil
}

func sendRequest(req *GrpcMainReq) error {
	ctx := context.Background()

	body, err := resolveBody(req.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	conf := newGrpcClientConfig(req)
	codec, err := buildCodec(ctx, conf, req.SubType, req.Method)
	if err != nil {
		return err
	}

	cc, err := NewGrpcClient(conf)
	if err != nil {
		return fmt.Errorf("dial server: %w", err)
	}
	defer cc.Close()

	count := req.Count
	if count <= 0 {
		count = 1
	}
	for i := range count {
		invokeReq, err := codec.CreateReqObj(body)
		if err != nil {
			return err
		}
		reply := codec.CreateReplyTarget()

		var header, trailer metadata.MD
		callOpts := []grpc.CallOption{grpc.Header(&header), grpc.Trailer(&trailer)}
		if opt := codec.GrpcCallOption(); opt != nil {
			callOpts = append(callOpts, opt)
		}

		err = cc.Invoke(ctx, normalizeMethodPath(req.Method), invokeReq, reply, callOpts...)
		if err != nil {
			return err
		}

		if count > 1 {
			fmt.Fprintf(os.Stderr, "--- request %d/%d ---\n", i+1, count)
		}
		printMetadataSection("header", header)
		printMetadataSection("trailer", trailer)

		out, err := codec.GetResponseOutput(reply)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			fmt.Fprintln(os.Stderr, "(empty response)")
		} else {
			_, err = os.Stdout.Write(out)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func newGrpcClientConfig(req *GrpcMainReq) *GrpcClientConfig {
	return &GrpcClientConfig{
		Host:                  req.Host,
		TLS:                   req.TLS,
		InitialConnWindowSize: req.InitialConnWindowSize,
		InitialWindowSize:     req.InitialWindowSize,
		Headers:               req.Headers,
	}
}

// Codec abstracts request construction, response parsing, and gRPC call options for sendRequest.
type Codec interface {
	// CreateReqObj builds the Invoke request value (proto.Message or []byte) from body.
	CreateReqObj(body []byte) (any, error)
	// CreateReplyTarget returns the reply target passed to Invoke (e.g. *dynamicpb.Message or *[]byte).
	CreateReplyTarget() any
	// GrpcCallOption returns an additional CallOption for the call, or nil to use the default proto codec.
	GrpcCallOption() grpc.CallOption
	// GetResponseOutput converts the Invoke reply into bytes to write to stdout; may return nil for empty responses.
	GetResponseOutput(resp any) ([]byte, error)
}

func buildCodec(ctx context.Context, conf *GrpcClientConfig, subType, method string) (Codec, error) {
	if subType == "" {
		inputMsg, outputMsgDesc, err := resolveMethodMessages(ctx, conf, method)
		if err != nil {
			return nil, err
		}
		// Only the descriptor is needed; a fresh instance is created in CreateReqObj.
		inputMsgDesc := inputMsg.ProtoReflect().Descriptor()
		return &protoJSONCodec{
			inputMsgDesc:  inputMsgDesc,
			outputMsgDesc: outputMsgDesc,
		}, nil
	}
	return &rawCodec{subType: subType}, nil
}

type protoJSONCodec struct {
	inputMsgDesc  protoreflect.MessageDescriptor
	outputMsgDesc protoreflect.MessageDescriptor
}

func (c *protoJSONCodec) CreateReqObj(body []byte) (any, error) {
	msg := dynamicpb.NewMessage(c.inputMsgDesc)
	if len(body) > 0 {
		if err := protojson.Unmarshal(body, msg); err != nil {
			return nil, fmt.Errorf("unmarshal proto-json request: %w", err)
		}
	}
	return msg, nil
}

func (c *protoJSONCodec) CreateReplyTarget() any {
	return dynamicpb.NewMessage(c.outputMsgDesc)
}

func (c *protoJSONCodec) GrpcCallOption() grpc.CallOption {
	return nil
}

func (c *protoJSONCodec) GetResponseOutput(resp any) ([]byte, error) {
	msg := resp.(*dynamicpb.Message)
	if proto.Size(msg) == 0 {
		return nil, nil
	}
	return protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg)
}

type rawCodec struct {
	subType string
}

func (c *rawCodec) CreateReqObj(body []byte) (any, error) {
	return body, nil
}

func (c *rawCodec) CreateReplyTarget() any {
	return &[]byte{}
}

func (c *rawCodec) GrpcCallOption() grpc.CallOption {
	return grpc.ForceCodec(NewRawCodec(c.subType))
}

func (c *rawCodec) GetResponseOutput(resp any) ([]byte, error) {
	out := *resp.(*[]byte)
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func buildDescriptorRegistry(fds []*descriptorpb.FileDescriptorProto) (*protoregistry.Files, error) {
	files := new(protoregistry.Files)
	fdByName := make(map[string]*descriptorpb.FileDescriptorProto, len(fds))
	for _, fd := range fds {
		fdByName[fd.GetName()] = fd
	}
	registered := make(map[string]bool)
	resolver := &fallbackResolver{primary: files, fallback: protoregistry.GlobalFiles}

	var register func(*descriptorpb.FileDescriptorProto) error
	register = func(fd *descriptorpb.FileDescriptorProto) error {
		if registered[fd.GetName()] {
			return nil
		}
		for _, dep := range fd.GetDependency() {
			if depFd, ok := fdByName[dep]; ok {
				if err := register(depFd); err != nil {
					return err
				}
			}
		}
		registered[fd.GetName()] = true
		fileDesc, err := protodesc.NewFile(fd, resolver)
		if err != nil {
			return fmt.Errorf("build file descriptor %s: %w", fd.GetName(), err)
		}
		return files.RegisterFile(fileDesc)
	}

	for _, fd := range fds {
		if err := register(fd); err != nil {
			return nil, err
		}
	}
	return files, nil
}

type fallbackResolver struct {
	primary  *protoregistry.Files
	fallback *protoregistry.Files
}

func (r *fallbackResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if fd, err := r.primary.FindFileByPath(path); err == nil {
		return fd, nil
	}
	return r.fallback.FindFileByPath(path)
}

func (r *fallbackResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if d, err := r.primary.FindDescriptorByName(name); err == nil {
		return d, nil
	}
	return r.fallback.FindDescriptorByName(name)
}

func printMetadataSection(section string, md metadata.MD) {
	if len(md) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "-- %s --\n", section)
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range md[k] {
			fmt.Fprintf(os.Stderr, "%s: %s\n", k, v)
		}
	}
	fmt.Fprintln(os.Stderr)
}

type GrpcClientConfig struct {
	Host                  string
	TLS                   bool
	InitialConnWindowSize int32
	InitialWindowSize     int32
	Headers               []string
}

func NewGrpcClient(conf *GrpcClientConfig) (*grpc.ClientConn, error) {
	creds := transportCredentials(conf.TLS)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	if conf.InitialConnWindowSize > 0 {
		opts = append(opts, grpc.WithInitialConnWindowSize(conf.InitialConnWindowSize))
	}
	if conf.InitialWindowSize > 0 {
		opts = append(opts, grpc.WithInitialWindowSize(conf.InitialWindowSize))
	}
	if len(conf.Headers) > 0 {
		md, err := parseHeaders(conf.Headers)
		if err != nil {
			return nil, err
		}
		opts = append(opts,
			grpc.WithUnaryInterceptor(outgoingMetadataUnaryInterceptor(md)),
			grpc.WithStreamInterceptor(outgoingMetadataStreamInterceptor(md)),
		)
	}
	return grpc.NewClient(conf.Host, opts...)
}

func outgoingMetadataUnaryInterceptor(md metadata.MD) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(contextWithOutgoingMetadata(ctx, md), method, req, reply, cc, opts...)
	}
}

func outgoingMetadataStreamInterceptor(md metadata.MD) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(contextWithOutgoingMetadata(ctx, md), desc, cc, method, opts...)
	}
}

func contextWithOutgoingMetadata(ctx context.Context, md metadata.MD) context.Context {
	if len(md) == 0 {
		return ctx
	}
	existing, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		return metadata.NewOutgoingContext(ctx, metadata.Join(existing, md))
	}
	return metadata.NewOutgoingContext(ctx, md.Copy())
}

func transportCredentials(tlsEnabled bool) credentials.TransportCredentials {
	if tlsEnabled {
		return credentials.NewTLS(&tls.Config{})
	}
	return insecure.NewCredentials()
}
