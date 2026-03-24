# Zeabur 镜像部署清单

## 镜像地址

优先使用固定版本：

```text
docker.io/aleiii/cli-proxy-api:aad3dfd9
```

如果你希望始终跟随最新版：

```text
docker.io/aleiii/cli-proxy-api:latest
```

## 创建服务

1. 进入 Zeabur 项目。
2. 选择 `Add Service`。
3. 选择 `Docker Images`。
4. 把上面的镜像地址填进去。

## 端口配置

最少只需要暴露主 API 端口：

```text
8317 / HTTP
```

如需在服务器上直接完成 OAuth / CLI 登录回调，再额外增加这些端口：

```text
8085
1455
54545
51121
11451
```

## 启动命令

保持留空：

```text
Start Command: 留空
```

## 配置文件读取规则

这版已经统一了配置路径，避免出现两个 `config.yaml` 不知道读哪个的问题。

不启用 SQL store 时：

```text
/CLIProxyAPI/config/config.yaml
```

启用 PostgreSQL 时：

```text
/CLIProxyAPI/data/pgstore/config/config.yaml
```

启用 MySQL 时：

```text
/CLIProxyAPI/data/mysqlstore/config/config.yaml
```

也就是说：

- 本地卷模式会读取 `/CLIProxyAPI/config/config.yaml`
- PostgreSQL 模式不会再读取 `/CLIProxyAPI/config/config.yaml`
- MySQL 模式不会再读取 `/CLIProxyAPI/config/config.yaml`

## 方案一：最简单的卷持久化

这是最省事的方式，适合先跑起来。

环境变量：

```env
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
CLIPROXY_CONFIG_PATH=/CLIProxyAPI/config/config.yaml
MANAGEMENT_PANEL_GITHUB_REPOSITORY=https://github.com/Dorixxx/clima.git
MANAGEMENT_BUNDLED_ASSET_MODE=replace
```

Volumes：

```text
/CLIProxyAPI/config
/CLIProxyAPI/data
/root/.cli-proxy-api
```

用途：

- `/CLIProxyAPI/config`：持久化主配置文件
- `/CLIProxyAPI/data`：持久化运行数据、静态资源、日志
- `/root/.cli-proxy-api`：持久化认证文件和 token 文件

## 方案二：PostgreSQL 持久化

适合把 `config` 和认证相关文件统一存到 PostgreSQL。

环境变量：

```env
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
PGSTORE_DSN=postgresql://user:pass@host:5432/cliproxy
PGSTORE_SCHEMA=public
PGSTORE_LOCAL_PATH=/CLIProxyAPI/data
MANAGEMENT_PANEL_GITHUB_REPOSITORY=https://github.com/Dorixxx/clima.git
MANAGEMENT_BUNDLED_ASSET_MODE=replace
```

Volumes：

```text
/CLIProxyAPI/data
/root/.cli-proxy-api
```

说明：

- 生效配置文件路径是 `/CLIProxyAPI/data/pgstore/config/config.yaml`
- 不需要再挂 `/CLIProxyAPI/config`
- `PGSTORE_LOCAL_PATH` 主要用于本地缓存和导出路径，建议保持 `/CLIProxyAPI/data`

## 方案三：MySQL 持久化

环境变量：

```env
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
MYSQLSTORE_DSN=user:pass@tcp(host:3306)/cliproxy?parseTime=true&charset=utf8mb4
MYSQLSTORE_DATABASE=cliproxy
MYSQLSTORE_LOCAL_PATH=/CLIProxyAPI/data
MANAGEMENT_PANEL_GITHUB_REPOSITORY=https://github.com/Dorixxx/clima.git
MANAGEMENT_BUNDLED_ASSET_MODE=replace
```

Volumes：

```text
/CLIProxyAPI/data
/root/.cli-proxy-api
```

说明：

- 生效配置文件路径是 `/CLIProxyAPI/data/mysqlstore/config/config.yaml`
- 不需要再挂 `/CLIProxyAPI/config`
- `MYSQLSTORE_LOCAL_PATH` 建议保持 `/CLIProxyAPI/data`

## 可直接复制的 Zeabur 最小配置

### 本地卷模式

```md
Image:
docker.io/aleiii/cli-proxy-api:aad3dfd9

Ports:
8317 / HTTP

Environment Variables:
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
CLIPROXY_CONFIG_PATH=/CLIProxyAPI/config/config.yaml
MANAGEMENT_PANEL_GITHUB_REPOSITORY=https://github.com/Dorixxx/clima.git
MANAGEMENT_BUNDLED_ASSET_MODE=replace

Volumes:
/CLIProxyAPI/config
/CLIProxyAPI/data
/root/.cli-proxy-api
```

### PostgreSQL 模式

```md
Image:
docker.io/aleiii/cli-proxy-api:aad3dfd9

Ports:
8317 / HTTP

Environment Variables:
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
PGSTORE_DSN=postgresql://user:pass@host:5432/cliproxy
PGSTORE_SCHEMA=public
PGSTORE_LOCAL_PATH=/CLIProxyAPI/data
MANAGEMENT_PANEL_GITHUB_REPOSITORY=https://github.com/Dorixxx/clima.git
MANAGEMENT_BUNDLED_ASSET_MODE=replace

Volumes:
/CLIProxyAPI/data
/root/.cli-proxy-api
```

### MySQL 模式

```md
Image:
docker.io/aleiii/cli-proxy-api:aad3dfd9

Ports:
8317 / HTTP

Environment Variables:
TZ=Asia/Shanghai
WRITABLE_PATH=/CLIProxyAPI/data
MYSQLSTORE_DSN=user:pass@tcp(host:3306)/cliproxy?parseTime=true&charset=utf8mb4
MYSQLSTORE_DATABASE=cliproxy
MYSQLSTORE_LOCAL_PATH=/CLIProxyAPI/data
MANAGEMENT_PANEL_GITHUB_REPOSITORY=https://github.com/Dorixxx/clima.git
MANAGEMENT_BUNDLED_ASSET_MODE=replace

Volumes:
/CLIProxyAPI/data
/root/.cli-proxy-api
```

## 部署后检查

部署完成后检查：

1. 服务日志中是否出现 `CLIProxyAPI Version`
2. 访问 Zeabur 分配的域名，确认 `8317` 可用
3. 首次启动后确认配置文件已经生成到当前模式对应的路径
4. 确认认证文件没有在重启后丢失

## 参考

- Zeabur 自定义镜像部署文档: https://zeabur.com/docs/zh-CN/deploy/customize-prebuilt
- Zeabur Volumes 文档: https://zeabur.com/docs/en-US/data-management/volumes
