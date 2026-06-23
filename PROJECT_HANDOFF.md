# CMDB 项目交接总结

更新时间：2026-06-15  
项目路径：`/Users/dongyasong/workspace/gin-postgresql-project`

## 1. 当前关键决策

### 机器唯一身份

- 机器生命周期以 `ipmi_ip` 作为稳定身份。
- `zbx_id` 会因为重装或异常自动变化，不能再作为判断机器是否存在的唯一依据。
- 后端 `POST /api/machine` 已改为按 `ipmi_ip` 幂等写入：
  - `ipmi_ip` 不存在：创建机器。
  - `ipmi_ip` 已存在：更新 IDC 信息，不再因为唯一键冲突返回 500。
  - 如果 `zbx_id` 变化，会同步更新 `machine_info`、`business_info`、`network_info` 里相同机器的 `zbx_id`。

### 未更新机器归档策略

采用“状态表 + 四张归档快照表”的方案：

- `machine_sync_state` 只记录机器同步状态，不存完整业务字段。
- 四张归档表保存机器被归档时的完整字段快照：
  - `archived_idc_info`
  - `archived_machine_info`
  - `archived_business_info`
  - `archived_network_info`
- `machine_archive_batches` 作为归档批次表，用于把四张归档表串起来。

生命周期：

1. 任意机器相关 `POST/PUT` 成功后刷新 `machine_sync_state.last_seen_at`。
2. 超过 24 小时未更新，状态改为 `stale`。
3. `stale` 后再经过 12 小时宽限期，复制四张主表快照到归档表，并从主表删除。
4. 归档保留 30 天。
5. 归档期内机器重新上报时，后端会尝试从归档快照恢复主表数据，再应用本次上报。
6. 恢复网络快照时，如果 IPv4/IPv6 已被其他记录占用，则跳过冲突网络记录，避免整台机器恢复失败。

### 前端方向

- 前端包管理切到 `pnpm`。
- UI 不使用组件库，使用 UnoCSS 原子类为主。
- 已重构：
  - 应用壳 `App.vue`
  - 登录页 `Login.vue`
  - 机器信息页 `MachineInfo.vue`
  - 新增归档机器页 `ArchiveMachines.vue`
- 旧页面暂时保留，通过 `src/styles/main.css` 提供兼容样式。

### Docker / OpenResty 方案

- 前端由 OpenResty 容器提供静态文件。
- OpenResty 反代 `/api/` 到后端 Go API。
- Compose 标准化为桥接网络：
  - 后端服务名：`golang-api`
  - OpenResty upstream：`golang-api:34185`
  - API 宿主机端口：`34185`
  - 前端宿主机端口：`34186`
- Docker 基础镜像已改用内网镜像代理前缀：
  - `1181.s.kuaicdn.cn:11818/docker.io/...`

## 2. 已完成的重要事项

### 后端

- 新增归档模型文件：`models/machine_archive.go`
- 新增归档处理器：`handlers/machine_archive.go`
- `database/postgres.go` 已加入 AutoMigrate：
  - `MachineSyncState`
  - `MachineArchiveBatch`
  - `ArchivedIDCInfo`
  - `ArchivedMachineInfo`
  - `ArchivedBusinessInfo`
  - `ArchivedNetworkInfo`
- 新增路由：
  - `GET /api/machine-archives`
  - `GET /api/machine-archives/:batch_id`
  - `GET /api/machine-sync-states`
- `main.go` 定时任务中加入：
  - `MarkStaleAndArchiveMachines()`
  - `CleanupExpiredMachineArchives()`
- `init-db.sql` 已补充新表和索引 DDL。
- `POST /api/machine` 已支持：
  - 按 `ipmi_ip` 幂等写入
  - 从归档自动恢复
  - 刷新机器同步状态
- `PUT /api/machine/:ipmi_ip` 已支持：
  - 同步相关表 `zbx_id`
  - 刷新机器同步状态
- `UpdateBusinessInfo`、`UpdateMachineInfo`、部分 `network_info` 写接口已刷新机器同步状态。
- `GetNetworkInfoStats` 已改为返回机房名称：
  - `idc_code`
  - `idc_name`
  - `count`

### 前端

- 新增 `frontend/pnpm-lock.yaml`。
- 新增 `frontend/uno.config.ts`。
- `frontend/vite.config.ts` 接入 UnoCSS。
- `frontend/src/main.ts` 接入：
  - `@unocss/reset/tailwind.css`
  - `virtual:uno.css`
- `frontend/src/App.vue` 重构为侧边栏 + 顶部栏工作台布局。
- `frontend/src/views/Login.vue` 重构为新登录页。
- `frontend/src/views/MachineInfo.vue` 重构为密集表格工作台。
- 新增 `frontend/src/views/ArchiveMachines.vue`：
  - 展示归档批次列表。
  - 点击批次可查看 IDC、硬件、业务、网络四张表快照。
- `frontend/src/router/index.ts` 新增 `/archives` 路由。
- `frontend/src/api/index.ts` 新增归档相关接口。
- `frontend/src/views/NetworkInfo.vue` 统计卡片已改为显示：
  - `机房编码 · 机房名称`
  - 副标题为 `机房网络记录数`

### Docker / 部署

- `Dockerfile` 基础镜像改为内网镜像代理。
- `Dockerfile.frontend`：
  - Node 镜像改为内网镜像代理。
  - OpenResty 镜像改为内网镜像代理。
  - 固定 `pnpm@9.15.0`。
  - 使用 `pnpm install --frozen-lockfile`。
- `openresty/nginx.conf`：
  - `/api/` 反代到 `golang-api:34185`。
- `docker-compose.yml`：
  - 去掉 `network_mode: host`。
  - API 映射 `34185:34185`。
  - 前端映射 `34186:80`。

## 3. 验证记录

最近已跑通过：

```bash
go test ./...
```

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm build
pnpm exec vue-tsc --noEmit
```

前端本地开发：

```bash
cd frontend
pnpm install
pnpm dev
```

默认访问：

```text
http://127.0.0.1:34187/
```

## 4. 数据库手工建表 SQL

程序有 AutoMigrate，但如果需要手动控制，可在 PostgreSQL 中执行 `init-db.sql` 里新增的表结构。

核心新增表：

```sql
CREATE TABLE IF NOT EXISTS machine_sync_state (
  ipmi_ip VARCHAR(16) PRIMARY KEY,
  zbx_id VARCHAR(50),
  status VARCHAR(20) NOT NULL,
  last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
  first_stale_at TIMESTAMP WITH TIME ZONE,
  archived_at TIMESTAMP WITH TIME ZONE,
  last_archive_batch_id VARCHAR(80),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS machine_archive_batches (
  archive_batch_id VARCHAR(80) PRIMARY KEY,
  ipmi_ip VARCHAR(16) NOT NULL,
  zbx_id VARCHAR(50),
  idc_code VARCHAR(10),
  idc_name VARCHAR(50),
  ssh_ip VARCHAR(16),
  archive_reason VARCHAR(40) NOT NULL,
  status VARCHAR(20) NOT NULL,
  last_seen_at TIMESTAMP WITH TIME ZONE,
  archived_at TIMESTAMP WITH TIME ZONE NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
  restored_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

四张归档明细表完整 DDL 已写入 `init-db.sql`。

首次上线建议补现有机器状态：

```sql
INSERT INTO machine_sync_state (ipmi_ip, zbx_id, status, last_seen_at, created_at, updated_at)
SELECT i.ipmi_ip, i.zbx_id, 'active', NOW(), NOW(), NOW()
FROM idc_info i
WHERE NOT EXISTS (
  SELECT 1 FROM machine_sync_state s WHERE s.ipmi_ip = i.ipmi_ip
);
```

## 5. 对现有脚本的影响

现有脚本不需要立刻改。

兼容原因：

- 原接口仍保留：
  - `POST /api/machine`
  - `PUT /api/machine/:ipmi_ip`
  - `PUT /api/machines/:ipmi_ip/business`
  - `PUT /api/machines/:ipmi_ip/machine-info`
  - `PUT /api/network-info/...`
- 原 JSON 字段未变。
- 后端新增的是状态刷新、幂等写入、归档恢复逻辑。

建议后续优化脚本：

- 判断新增或更新时，以 `ipmi_ip` 是否存在为准。
- 不要再用旧 `zbx_id` 判断机器是否存在。

## 6. 当前待办事项

### 高优先级

- 确认生产数据库是否手动执行新增表 DDL。
- 确认 OpenResty 镜像代理路径是否可拉：
  - `1181.s.kuaicdn.cn:11818/docker.io/openresty/openresty:1.25.3.1-0-alpine`
  - 如果不可拉，需要替换为公司内部已有 OpenResty 镜像。
- 在测试环境用真实数据验证归档流程：
  - 手动构造 `last_seen_at` 超时。
  - 验证 `stale`。
  - 验证归档后四张主表删除、四张归档表保留。
  - 验证重新上报后恢复。
- 检查 `network_info` 单条/批量写接口是否全部刷新同步状态，目前已覆盖主要路径，但建议再全面审计一遍。

### 中优先级

- 把 `IDCInfo.vue`、`NetworkInfo.vue`、`BusinessInfo.vue`、`Deletion.vue` 继续统一重构成 UnoCSS 风格。
- 给归档功能增加管理员手动操作：
  - 手动归档某台机器。
  - 手动恢复某台机器。
  - 手动清理某批归档。
- 给归档列表增加导出 Excel。
- 给 `machine_sync_state` 做前端页面，展示 active / stale / archived 状态。

### 低优先级

- 清理仓库里已跟踪的 `frontend/node_modules`、`frontend/dist`，建议后续加 `.gitignore` 并从 Git 跟踪中移除。
- 整理 `restart_gogin.sh`，当前它有历史本地改动，不是本次主要任务。
- Swagger 文档尚未同步新增接口。

## 7. 重要文件修改记录

### 后端核心

- `models/machine_archive.go`
  - 新增机器同步状态、归档批次、归档快照模型。
- `handlers/machine_archive.go`
  - 新增归档、恢复、清理、查询接口逻辑。
- `handlers/machine.go`
  - `CreateMachine` 改为按 `ipmi_ip` 幂等写入。
  - `UpdateMachine` 同步相关表 `zbx_id`。
  - 写接口刷新同步状态。
- `handlers/network_info.go`
  - 网络写接口刷新机器同步状态。
  - 网络统计接口补 `idc_name`。
- `database/postgres.go`
  - AutoMigrate 新归档模型。
  - 启动时给已有机器补 `machine_sync_state`。
- `main.go`
  - 新增归档 API 路由。
  - 定时任务加入归档扫描和清理。
- `init-db.sql`
  - 新增归档相关 DDL 和索引。

### 前端核心

- `frontend/package.json`
  - 增加 UnoCSS 相关依赖。
  - 使用 `pnpm`。
- `frontend/pnpm-lock.yaml`
  - 新增 pnpm 锁文件。
- `frontend/uno.config.ts`
  - UnoCSS 配置和快捷类。
- `frontend/src/App.vue`
  - 重构整体布局。
- `frontend/src/views/Login.vue`
  - 重构登录页。
- `frontend/src/views/MachineInfo.vue`
  - 重构机器信息页面。
- `frontend/src/views/ArchiveMachines.vue`
  - 新增归档机器页面。
- `frontend/src/views/NetworkInfo.vue`
  - 修改统计卡片显示真实机房名称。
- `frontend/src/styles/main.css`
  - 保留基础样式和旧页面兼容层。

### Docker / OpenResty

- `Dockerfile`
  - Go / Alpine 基础镜像改为内网代理前缀。
- `Dockerfile.frontend`
  - Node / OpenResty 基础镜像改为内网代理前缀。
  - 固定 pnpm 版本。
- `docker-compose.yml`
  - 改成桥接网络 + 端口映射。
- `openresty/nginx.conf`
  - `/api/` 反代到 `golang-api:34185`。

## 8. 当前注意事项

- `restart_gogin.sh` 有本地历史改动，之前未处理。
- `frontend/dist` 和 `frontend/node_modules` 在仓库中已有跟踪/生成物噪音，后续建议专门整理。
- 若生产只允许 Docker，需要优先确认镜像代理是否包含 OpenResty 镜像。
- 归档功能上线初期建议先缩短扫描周期或手动改时间测试，不要直接等 24h + 12h。
