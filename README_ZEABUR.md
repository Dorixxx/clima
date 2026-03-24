# Zeabur 部署清单

## 1. 创建服务

1. 进入 Zeabur 项目。
2. 选择 `Add Service`。
3. 选择 `Docker Images`。
4. 镜像地址填写：

```text
docker.io/aleiii/cli-proxy-api:latest
```

## 2. 端口配置

只需要先暴露主 API 端口：

- `Port`: `8317`
- `Protocol`: `HTTP`

说明：

- `8317` 是主服务端口。
- `8085`、`1455`、`54545`、`51121`、`11451` 更像 OAuth/CLI 登录回调端口。
- 如果你要在 Zeabur 上直接完成 OAuth 登录，再额外加这些端口。

## 3. 启动命令

留空即可：

```text
Start Command: 留空
```

## 4. 环境变量

先用最稳的本地卷持久化方案：

```env
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
CLIPROXY_CONFIG_PATH=/CLIProxyAPI/config/config.yaml
```

## 5. 持久化挂载

在 Zeabur 的 Volumes 里添加 3 个挂载：

```text
/CLIProxyAPI/config
/CLIProxyAPI/data
/root/.cli-proxy-api
```

用途：

- `/CLIProxyAPI/config`：保存 `config.yaml`
- `/CLIProxyAPI/data`：保存日志、面板静态资源和运行时数据
- `/root/.cli-proxy-api`：保存认证文件

## 6. PostgreSQL 持久化方案

如果你想把 `config` 和 `auth token` 存到 PostgreSQL，再加这些环境变量：

```env
PGSTORE_DSN=postgresql://user:pass@host:5432/cliproxy
PGSTORE_SCHEMA=public
PGSTORE_LOCAL_PATH=/CLIProxyAPI/data
```

建议仍保留：

```text
/CLIProxyAPI/data
```

## 7. MySQL 持久化方案

如果你想改用 MySQL，再加这些环境变量：

```env
MYSQLSTORE_DSN=user:pass@tcp(host:3306)/cliproxy?parseTime=true&charset=utf8mb4
MYSQLSTORE_DATABASE=cliproxy
MYSQLSTORE_LOCAL_PATH=/CLIProxyAPI/data
```

建议仍保留：

```text
/CLIProxyAPI/data
```

## 8. 部署后检查

部署完成后检查：

1. 服务是否启动成功。
2. 访问分配的域名或 `8317` 端口。
3. 查看日志里是否出现 `CLIProxyAPI Version`。
4. 首次启动后确认配置文件已生成到挂载目录。

## 9. 推荐的最简配置

直接照这个填就可以：

```md
Image:
docker.io/aleiii/cli-proxy-api:latest

Port:
8317 / HTTP

Environment Variables:
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
CLIPROXY_CONFIG_PATH=/CLIProxyAPI/config/config.yaml

Volumes:
/CLIProxyAPI/config
/CLIProxyAPI/data
/root/.cli-proxy-api
```

## 参考

- Zeabur 自定义镜像部署文档: https://zeabur.com/docs/zh-CN/deploy/customize-prebuilt
- Zeabur Volumes 文档: https://zeabur.com/docs/en-US/data-management/volumes
