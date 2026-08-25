基于 Go 实现的光伏电站逆变器群控与并网管理系统项目，一款新能源设备控制服务，完成逆变器启停、并/离网切换、功率调度与故障恢复。

# PVInverterControl

PVInverterControl 是一个自包含的光伏电站逆变器群控与并网管理服务。中控按辐照与
负荷计算功率目标，经通讯层向各逆变器下发限功率、启停、参数与并网切换指令；
逆变器状态机与并网状态机受控流转，通讯链路异常时进入保护停机，恢复后重新并网。

## 构建

```bash
go build -mod=vendor ./...
```

## 运行

```bash
go run -mod=vendor ./cmd/pvcontrol -addr 127.0.0.1:8090
```

启动后访问 http://127.0.0.1:8090/ 打开控制台页面。

## HTTP 接口

- `POST /api/v1/irradiance` 更新辐照并刷新功率目标
- `GET /api/v1/inverters` 查询逆变器状态
- `POST /api/v1/connect/{id}` 逆变器并网
- `POST /api/v1/start/{id}` 启动逆变器
- `POST /api/v1/stop/{id}` 停止逆变器
- `POST /api/v1/recover/{id}` 故障恢复并重新下发限功率
- `POST /api/v1/patrol` 通讯巡检
- `POST /api/v1/param` 参数下发
- `GET /api/v1/alarms` 查询告警
- `GET /healthz` 健康检查

## 运行模型

服务内置模拟逆变器通讯链路，默认注册三台不同型号的逆变器，中控按辐照计算功率
目标并统一下发；通讯、参数、告警与去重组件全部在服务内真实联动。
