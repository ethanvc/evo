
# 典型场景思考

## gateway通过json调用后端grpc服务，会返回error和json格式的回包，如何生成一个Error，用于监控上报？
极端场景下会出现error类型和resp中都包含错误码的情况，因此转成Error需要参考两者的信息。

一般，一个服务的接口的返回包具有相同的schema，可以用一个统一的方式提取错误码、错误信息。
因为后端服务比较多，可能需要动态配置比较好。
但是动态配置有个复用问题（不同国家和环境）,导致运维成本非常高。



## 如何将path转成pattern进行监控上报？
处理两种情况：
1. path中包含参数。
2. path是统一接入API，需要根据header或特定的参数进行分发。此时仅上报path统计意义不大。

### 场景：不同环境域名不同，希望一次配置都生效
https://api.test.xx.com/api/get
https://api.xx.com/api/get

### 场景：路径包含参数
https://api.xx.com/api/get/{student-id}
https://api.xx.com/api/get/{student-id}/photos/{*photo-path}

### 场景：统一接入api
https://openapi.alipay.com/gateway.do
通过body中的method标记是调用的哪个方法。

### 解决思路：
1. 使用radix tree提取模式上报。无法获取的，就告警然用户添加。
2. 这种情况不紧急，可以在代码中设置上报的method。随版本发布。
3. 采用代码中处理，好处是运维方便。可以抽取到一个仓库，这样可以复用规则。

## 日志敏感信息如何处理？

## 配置的运维问题如何解决？环境、国家太多，互相隔离，但是这里的配置是可以保持一致的。