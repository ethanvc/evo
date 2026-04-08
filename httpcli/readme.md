## 拦截器的参数是用url.URL还是 url string?
最终倾向于传string。
1. 标准库接收string类型。
2. url格式不对的时候，无法生成合法的url.URL对象，此时，无法进入拦截器执行逻辑。
   如果允许nil值进入拦截器，则所有拦截器都要处理这个特殊case，容易引发panic。