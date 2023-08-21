如果你是针对已存在的 proto 文件，修改 message 或者 service 的定义或者注释，那么可以移步到**步骤三**。

# 步骤一：确保 proto 文件的路径满足要求

假设你想新建一个服务，并为这个服务创建了一个 proto 文件用以定义 API。那么在生成代码之前，请确保该 proto 文件位于项目顶层目录下的 api 目录中，并按照规定的次级目录定义放置这个 proto 文件。  
规定的次级目录定义为：`{服务名}/service/{版本号}/{proto文件}`

- 目前开发阶段要求的版本号暂定都是 v1

例如 user 服务的 user.proto 文件，那么相对于项目顶层目录的路径就应该是 `api/user/service/v1/user.proto`

![image.png](../../img/proto/api_proto_location.png)

如果是服务内的 proto 文件，由于一般服务内涉及到的 proto 文件只与服务的配置相关，所以这里就以此来展开。请确保该 proto 文件位于项目顶层目录下的 app 目录中，并按照规定的次级目录定义放置这个 proto 文件。
规定的次级目录定义为：`{服务名}/service/internal/conf/{proto文件}`  
例如 user 服务配置的 conf.proto，那么相对于项目顶层目录的路径就应该是 `app/user/service/internal/conf/conf.proto`

![image.png](../../img/proto/app_proto_location.png)

同样可以使用 kratos 的 CLI 工具来生成 proto 文件, `kratos proto add {服务名}/service/{版本号}/{服务}.proto`. [详情](https://go-kratos.dev/docs/getting-started/usage#%E6%B7%BB%E5%8A%A0-proto-%E6%96%87%E4%BB%B6)

# 步骤二：确保 proto 文件内的包定义满足要求

假如是一个服务的 API 定义，`package`要定义为`{服务名}.service.{版本号}`，同时，必须定义`go_package`，并且要定义为`github.com/toomanysource/atreus/api/{服务名}/service/{版本号};{版本号}`  
同样以 user 服务的 user.proto 文件为例：

- `package`定义为`user.service.v1`
- `go_package`定义为`github.com/toomanysource/atreus/api/user/service/v1;v1`

```protobuf
// ...
package user.service.v1;

option go_package = "github.com/toomanysource/atreus/api/user/service/v1;v1";
// ...
```

如果是服务内的 proto 配置文件，`package`要定义为`{服务名}.service.internal.conf` ，同时，必须定义`go_package`，并且要定义为`github.com/toomanysource/atreus/app/{服务名}/service/internal/conf;conf`  
同样以 user 服务配置的 conf.proto 文件为例：

- `package`定义为`user.service.internal.conf`
- `go_package`定义为`github.com/toomanysource/atreus/app/user/service/internal/conf;conf`

```protobuf
// ...
package user.service.internal.conf;

option go_package = "github.com/toomanysource/atreus/app/user/service/internal/conf;conf";
// ...
```

如果使用生成工具，`go_package` 由 `go.mod` 中的 module 决定，而 `package` 则由生成proto文件的路径决定。
另外，在 proto文件的 `go_package` 中，分号后面是定义生成文件的`package`。 例如`v1;v1`，则生成文件的 package 为 `package v1`.
# 步骤三：生成代码

**注意前提环境：启用 Docker**

如果你已经把前面的两个步骤完成了，或者你只是修改 message 或者 service 的定义或者注释，那么你可以直接执行

```bash
make proto
```

这个命令会先构建生成 proto 代码的镜像并用此镜像启动一个容器，用以生成 api 和 app 目录下的全部 proto 文件的代码，并将这些代码文件跟 proto 文件放置在同一个目录下。

- 生成的代码：
  - `.pb.go`
  - `_grpc.pb.go`（如果定义了 service）
  - `_http.pb.go`（如果在 service 中定义了 http api)

# 步骤四：编写其余的代码

到这里你就可以愉快地编写代码了～ :)
