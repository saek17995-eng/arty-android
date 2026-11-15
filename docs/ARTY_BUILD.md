# Arty 构建与发布说明

Arty 是 DateChat 的品牌重命名版本，继续使用原有 Go 后端、腾讯云 IM、WebRTC/TRTC 配置和接口。

## Android APK

GitHub Actions 工作流 `.github/workflows/android-apk.yml` 会按 Flutter 稳定版生成 Android 平台壳，构建 `com.arty.arty` 的 ARM64 release APK，默认显示名称为 `Arty`，默认后端为 `https://hs2.rqez.cn`。

手动构建时可在 Actions 中覆盖：

- `api_base_url`：Arty 后端地址；
- `turn_url`、`turn_username`、`turn_credential`：跨公网 WebRTC 所需的 TURN 服务；
- `app_name`：手机桌面显示名称。

## 发布包

推送 `v*` 标签后，工作流会将 `arty-*.apk` 附加到 GitHub Release。源码仓库本身包含完整的客户端、服务端、迁移脚本和构建工作流；密钥、`.env`、本地数据库和临时构建目录不会提交。

## 许可证与来源

Arty 本次改造只重命名 DateChat 客户端品牌，不将 Element X 源码或二进制混入运行时。若今后重新引入 Element X 代码，必须保留其 AGPL-3.0 许可证、版权声明及对应源码提供义务，并单独完成许可证审查。
