# 发送grpc请求
可以通过curl完成：
```aiignore
no_proxy='*' curl --http2-prior-knowledge -XPOST -d'{}' http://host_ip_or_domain:80 -v -H 'Content-Type:application/grpc'
```