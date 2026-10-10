# 内核两轮优化实施记录（2026-10-10）

本次同时修改 SDM 客户端内核、配套 Xray-core 的 SSPlus UDP 扩展和客户端构建脚本。保留现有节点配置方式、`auth_aes128_md5`、`tls1.2_ticket_auth`、UoT v1/v2 和原生 UDP v3。原有功能标签继续保留。

## 已实现

| 范围 | 实现与效果 |
| --- | --- |
| SSPlus TCP | 同步合并握手写入，读响应前及返回连接前立即刷新，不增加计时等待；plain 握手底层写入 2→1 次，TLS-ticket 握手 4→2 次。 |
| SSR 与混淆 | 读写方向各自复用帧、密钥和摘要缓冲；同一帧复用 HMAC。TLS 外形记录保持 4096 字节，最多合并四条已就绪记录后一次写出。 |
| UDP 回退 | 原生建立期间，250 ms 后并行建立 UoT；原生提前失败则立即回退。成功通道胜出，取消并关闭另一通道，不重放业务报文。未知节点只允许一次并发原生探测，失败冷却 30 秒，切换网络清除能力结果。 |
| UDP 分配与包数 | 减少地址/数据复制，单分片直接交付。加入经认证且验证路径后的 MTU1 紧凑报文；常见地址的 1200 字节业务载荷由两包减为一包。 |
| DNS | 恢复正常 TTL，过期五分钟内可先返回旧值并合并后台刷新；超过宽限期不再使用旧值。 |
| 流量统计 | 活跃连接绑定原子计数器，读写回调直接累加；清零保留计数器对象，既有连接继续正确计数。 |
| 内存采样 | 状态接口与 Clash `/memory` 共用每秒一次的 `runtime/metrics` 采样，读取内存无需枚举连接。 |
| 历史连接 | 关闭时释放路由规则及嗅探工作对象引用，保留 UI 所需字段；可配置历史条数。 |
| 状态轮询 | 成功启动后缓存当前配置名，避免状态轮询反复打开 LevelDB。 |
| 内存预算 | 按设备物理内存设置 Go 软限制预算，保留现有关闭限制/显式调试配置入口。 |
| 延迟指标 | 统一首请求计时，记录 dial、first_request、warm_request；只有实际复用连接的第二次请求才标记 warm。 |
| 体积与构建 | 诊断代码复用已有 sagernet/quic-go，去掉第二套 QUIC 的直接依赖；Windows 与 Android 构建支持显式 PGO profile，默认关闭。 |

## 实测

Windows/amd64、Intel i5-12400F，同机回环同步 UDP echo，包含客户端、服务端和测试路由开销。优化前基线取修改前的 Git index；每项运行三次，以下时间取中间值附近结果。它们不是公网吞吐或手机 RSS 测试，小幅耗时差异不应视为稳定网络收益。

| 场景/载荷 | 优化前 ns/op | 优化后 ns/op | 优化前 B/op | 优化后 B/op | 优化前 allocs/op | 优化后 allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| native / 64 B | 46504 | 45168 | 1273 | 1016 | 34 | 28 |
| native / 1200 B | 53271 | 51511 | 3781 | 3418 | 38 | 28 |
| UoT v1 / 64 B | 49456 | 46966 | 21298 | 19606 | 83 | 55 |
| UoT v1 / 1200 B | 60062 | 57454 | 26036 | 19648 | 83 | 55 |
| UoT v2 / 64 B | 49900 | 49851 | 21450 | 19823 | 86 | 58 |
| UoT v2 / 1200 B | 61417 | 57992 | 26293 | 19829 | 86 | 58 |

UoT 1200 B 的分配字节约减少 25%，分配次数约减少 33%。统计微基准：原先已有节点累加约 40.18 ns、24 B、2 次分配；查找复用后约 15.20 ns、0 分配；实际回调使用的绑定指针约 6.802 ns、0 分配。

基准命令（在 `sdm-sing-box` 运行）：

```powershell
go test ./protocol/ssplus -run '^$' -bench '^BenchmarkUDPPolicy$' -benchmem -benchtime=400ms -count=3
go test ./experimental/clashapi/trafficontrol -run '^$' -bench . -benchmem
```

## 配置与兼容性

现有 SSPlus 节点无需新增字段。MTU1 仅在新客户端收到新服务端的认证能力提示，且 1360 字节双向路径验证成功后启用。连接旧 v3 服务端保持原 1200 字节上限，旧客户端也能连接新服务端。TCP 与 UoT 格式不变；XrayR 的节点配置无需为本次优化增加字段。

MTU1 不属于持续动态 PMTUD，已有会话中途路径 MTU 下降仍可能丢包，需要上层重建会话。UoT 并发回退只发生在会话建立阶段，不迁移已经发送业务数据的会话。

关闭连接历史默认仍保留 1000 条，完整内核 JSON 可按需设置，例如只保留 200 条；设为 0 关闭历史：

```json
{
  "experimental": {
    "clash_api": {
      "closed_connections_limit": 200
    }
  }
}
```

内存预算为物理内存的 1/16：Android 限定在 128–512 MiB，桌面限定在 256–1024 MiB。libbox 使用预算的 3/4 设置 Go 软限制，即 Android 96–384 MiB、桌面 192–768 MiB。iOS 保留原有 NetworkExtension 策略。该值不是整个进程 RSS 硬上限，也不包含 Flutter、socket 和全部 native 内存；不能据此宣称实际占用下降到这些数值。

UI 内存数值改为 Go 管理的已提交内存（total minus released），采样最多缓存一秒，不等同进程 RSS。延迟历史增加 `dial_delay`、`first_request_delay`、`warm_request_delay`；dial 只表示 DialContext 的耗时，不包含某些协议延后到第一次写入的握手。未扩展现有 UI protobuf。

## 构建与 PGO

构建脚本为应用目录的 `scripts/build_protocol_core.ps1`，Windows 与 Android 继续使用原来的标识符混淆和功能标签。普通构建默认 `-pgo=off`。有代表真实业务的 CPU pprof 后可执行：

```powershell
./scripts/build_protocol_core.ps1 -Platform windows -PGOProfile C:/profiles/windows-cpu.pprof
./scripts/build_protocol_core.ps1 -Platform android -PGOProfile C:/profiles/android-cpu.pprof
```

脚本验证 profile 可读取，记录路径及 SHA256，并在退出时恢复调用方环境变量。没有用合成基准 profile 开启生产 PGO，因此本记录不声称 PGO 性能收益。Android 原有发布流程已分 ABI，双 ABI AAR 大小不能直接解释为单 ABI APK 大小。

正式混淆构建均成功，PGO 为 off；产物已同步到应用打包目录并验证 SHA256 一致。以下与本次构建前目录中的正式产物比较：

| 产物 | 优化前字节 | 优化后字节 | 变化 | 应用使用路径 |
| --- | ---: | ---: | --- | --- |
| Windows DLL | 40060416 | 38974976 | 减少 1085440 字节，约 2.71% | `F:/pro/flutter/sdm-app/windows/libs/sdm-c.dll` |
| Android 双 ABI AAR | 30252798 | 30294882 | 增加 42084 字节，约 0.14% | `F:/pro/flutter/sdm-app/android/app/libs/libbox.aar` |

Windows SHA256：`4F7F0DA0AEA9AE1C19BE266EDABC329EC3787B0F0F21EA8BADFFB98B21D7DD89`。

Android SHA256：`65FD3C74CDFA18419178CF062B36250188E55B3F902488197739302F00DC89F2`。AAR 已确认包含 `jni/arm64-v8a/libbox.so`（41857672 字节）、`jni/x86_64/libbox.so`（44303752 字节）和 Java bindings。Android 本次没有包体缩减。

内核目录也保留 `bin/sdm-core-vless-ssplus.dll`、`bin/sdm-c.dll`、`bin/libbox-vless-ssplus.aar` 和两份 `core-build-*.json` 构建清单。本次构建的是内核库，没有重新打包完整 Flutter 安装包；Xray 服务端源码已更新并完成测试构建，没有部署或重启运行中的服务。

## 验证范围

已通过客户端 SSPlus、UDP 策略、URLTest、DNS、统计、monitoring 的 race 测试，以及根模块 config/hcore 的生产标签 race 测试。覆盖：回退取消与冷却、未知节点探测合并、统计清零后持续计数、当前配置切换、DNS 并发刷新合并与宽限期、握手写入合并、MTU1 协商及丢弃大包路径回退。

跨进程互通已验证新客户端↔新 Xray、旧客户端→新 Xray、新客户端→旧 v3 Xray。服务端 SSPlus 同样运行 race 回归。未运行手机真机功耗、峰值 RSS 和公网弱网长期压测。

回归命令（分别在对应模块目录运行）：

```powershell
# sdm-core
go test -race -ldflags=-checklinkname=0 -tags 'with_gvisor,with_quic,with_utls,with_clash_api,with_grpc,with_purego' ./v2/config ./v2/hcore ./cmd
# sdm-sing-box
go test -race ./transport/ssplus/... ./protocol/ssplus ./common/urltest ./common/memorystats ./experimental/clashapi/trafficontrol ./common/monitoring ./daemon ./dns
# Xray-core-main
go test -race ./proxy/ssplus/...
```
