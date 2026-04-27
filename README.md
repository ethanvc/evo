# 简介

# 待跟进
1. header的名单机制，合并打印日志的处理方式。
2. 增加print-json-if-posiable。

## event规范
打印日志需要提供一个event参数，这个参数需要满足如下规范：
1. 大驼峰格式。
2. 简短的英文描述，能够直观的说明问题。

## 错误
1. 错误均使用xobs.Error表示。
2. 在领域边界处将error转换成xobs.Error。
