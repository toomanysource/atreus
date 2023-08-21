# api下的proto文件必须加注释

- api下的proto文件是某个服务对外提供的请求接口定义，必须有良好的注释来帮助自己和其他开发者。
- 而对于app下的proto文件，由于都属于是配置文件，所以在定义不复杂的情况下，允许不添加注释。
- 
# 注释请使用中文表达

不是要求不能出现英文，而是说明信息应该用中文来表述。例如
```protobuf
// SomeService 提供一些其他服务请求的接口
service SomeService {
  // ...
}
```

# 尽量使用`//`而不是`/* */`来作为注释符号

# 避免在`//`注释符号之间添加空行

- 一个不好的示例
```protobuf
// 一些关于该service的注释
service BadCase {
  // XXXXXX
  //
  // XXXXXX
  field BAD
}
```

- 一个好的示例
```protobuf
// 一些关于该service的注释
service GoodCase {
  // XXXXXX
  // XXXXXX
  field GOOD
}
```
