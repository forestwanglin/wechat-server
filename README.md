# 微信公众号服务器
> 用以微信公众号的后端，提供登录验证功能

## 功能
+ [x] Access Token 自动刷新 & 提供外部访问接口
+ [x] 自定义菜单（需要你的公众号有这个权限）
+ [x] 登录验证
+ [ ] 自定义回复

## 展示
![demo1](https://user-images.githubusercontent.com/39998050/200124147-3338a2eb-8193-4068-ae6f-276cfe16a708.png)
![demo2](https://user-images.githubusercontent.com/39998050/200124177-78636b4c-0aac-4860-a138-68f3d92477b9.png)

## 部署
### 手动部署
1. 从 [GitHub Releases](https://github.com/songquanpeng/wechat-server/releases/latest) 下载可执行文件或者从源码编译：
   ```shell
   git clone https://github.com/songquanpeng/wechat-server.git
   go mod download
   go build -ldflags "-s -w" -o wechat-server
   ````
2. 运行：
   ```shell
   chmod u+x wechat-server
   ./wechat-server --port 3000 --log-dir ./logs
   ```
3. 访问 [http://localhost:3000/](http://localhost:3000/) 并登录。初始账号用户名为 `root`，密码为 `123456`。

更加详细的部署教程[参见此处](https://iamazing.cn/page/how-to-deploy-a-website)。

### 基于 Docker Compose 进行部署
仓库提供了 `compose.yaml`，默认使用 SQLite，并将数据和上传文件持久化到宿主机的 `/opt/wechat-server/data`：

```shell
cp .env.example .env
# 编辑 .env，至少设置 SESSION_SECRET
openssl rand -hex 32

docker compose up -d --build
docker compose ps
docker compose logs -f wechat-server
```

应用只绑定到宿主机 `127.0.0.1:13304`，建议由宿主机 Nginx 终止 HTTPS 并反向代理到该端口。需要更换端口或数据目录时，修改 `.env` 中的 `WECHAT_PORT` 或 `WECHAT_DATA_DIR`。

如果服务器已经有 MySQL 或 Redis，按 `.env.example` 设置 `SQL_DSN`、`REDIS_CONN_STRING` 后重建容器即可；容器访问宿主机服务使用 `host.docker.internal`，Compose 已配置对应的 `extra_hosts`。

旧的单容器方式仍然可用：`docker run -d --restart always -p 3000:3000 -v /home/ubuntu/data/wechat-server:/data justsong/wechat-server`。

## 配置
1. 从 [GitHub Releases](https://github.com/songquanpeng/wechat-server/releases/latest) 下载可执行文件。
2. 系统本身开箱即用，有一些环境变量可供配置：
   1. `REDIS_CONN_STRING`: 设置之后，将启用 Redis，例如 `redis://:密码@主机:6379/0`。
      Redis 用于登录会话存储、接口限流，以及启用系统统计功能时保存相关状态。未设置时，会话使用加密 Cookie，限流使用进程内实现；单实例可以不配置 Redis。
   2. `SESSION_SECRET`: 设置之后，将使用给定会话密钥。建议使用 `openssl rand -hex 32` 生成 32 字节随机值，例如 `SESSION_SECRET=...`。生产环境应固定该值，不能每次启动都更换，否则已有登录会话会失效。
   3. `SQL_DSN`: 设置之后，将使用 MySQL 而非 SQLite，例如 `user:password@tcp(host:3306)/wechat_server?charset=utf8mb4&parseTime=True&loc=Local`。需要提前创建数据库 `wechat_server`，应用启动时会自动创建和升级表结构，但不会创建数据库本身。
3. 运行: 
   1. `chmod u+x wechat-server`
   2. `./wechat-server --port 3000`
4. 初始账户用户名为 `root`，密码为 `123456`，记得登录后立刻修改密码。
5. 前往[微信公众号配置页面 -> 设置与开发 -> 基本配置](https://mp.weixin.qq.com/)获取 AppID 和 AppSecret，并在我们的配置页面填入上述信息，另外还需要配置 IP 白名单，按照页面上的提示完成即可。
6. 前往[微信公众号配置页面 -> 设置与开发 -> 基本配置](https://mp.weixin.qq.com/)填写以下配置：
   1. `URL` 填：`https://<your.domain>/api/wechat`
   2. `Token` 首先在我们的配置页面随便填写一个 Token，然后在微信公众号的配置页面填入同一个 Token 即可。
   3. `EncodingAESKey` 点随机生成，然后在我们的配置页面填入该值。
   4. 消息加解密方式选择明文模式。
7. 之后保存设置并启用设置。
8. 当前版本需要重启服务才能应用配置信息，因此请重启服务。

## API
### 获取 Access Token
1. 请求方法：`GET`
2. URL：`/api/wechat/access_token`
3. 无参数，但是需要设置 HTTP 头部：`Authorization: <token>`

### 通过验证码查询用户 ID
1. 请求方法：`GET`
2. URL：`/api/wechat/user?code=<code>`
3. 需要设置 HTTP 头部：`Authorization: <token>`

### 注意
需要将 `<token>` 和 `<code>` 替换为实际的内容。