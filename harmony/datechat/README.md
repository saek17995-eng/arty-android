# DateChat HarmonyOS

DateChat 的原生 HarmonyOS（ArkTS/ArkUI）工程。当前工程包含原生登录界面、短信验证码与登录接口，以及腾讯云 IM 和 TRTC 的 HarmonyOS SDK 依赖。

## 环境要求

- DevEco Studio / Command Line Tools 6.0 或更高版本
- HarmonyOS SDK API 17 或更高版本（当前目标 API 22，最低兼容 API 17）
- JDK 17

## 本地配置

`local.properties` 只用于本机 SDK 路径，不应提交。登录页中的后端地址默认是 `http://192.168.1.102:8080`，安装到真机前请改成手机能够访问的 HTTPS 或局域网地址。

## 构建

```sh
ohpm install
hvigorw --mode module -p product=default -p module=entry@default -p buildMode=debug assembleHap
```

未配置签名时，输出位于：

```text
entry/build/default/outputs/default/entry-default-unsigned.hap
```

## 真机签名

华为商用 HarmonyOS 设备只接受有效的 HarmonyOS 应用签名。请先完成华为开发者实名认证，然后在 DevEco Studio 中登录相同账号，连接目标设备并开启自动签名。签名配置会生成开发证书与包含设备 UDID 的调试 Profile；这些材料不要提交到仓库。

