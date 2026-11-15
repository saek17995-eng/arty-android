import axios, { AxiosError, AxiosResponse } from '@ohos/axios';
import { promptAction } from '@kit.ArkUI';

interface ApiEnvelope<T> {
  code: number;
  msg: string;
  data: T;
}

interface SmsCodeData {
  debug_code?: string;
  expire_in: number;
  resend_in: number;
}

interface TokenData {
  access_token: string;
  refresh_token: string;
}

interface UserData {
  uid: number;
  nickname: string;
}

interface ImData {
  enabled: boolean;
  sdk_app_id: number;
  user_id: string;
  usersig: string;
}

interface LoginData {
  uid: number;
  token: TokenData;
  user: UserData;
  im: ImData;
}

@Entry
@Component
struct Index {
  @State serverUrl: string = 'http://192.168.1.102:8080';
  @State phone: string = '';
  @State code: string = '';
  @State busy: boolean = false;
  @State loggedIn: boolean = false;
  @State nickname: string = '';
  @State serviceStatus: string = '原生 HarmonyOS 客户端';

  build() {
    Column() {
      Row() {
        Column() {
          Text('DateChat')
            .fontSize(30)
            .fontWeight(FontWeight.Bold)
            .fontColor('#202124')
          Text('HarmonyOS')
            .fontSize(14)
            .fontColor('#6A6F7A')
            .margin({ top: 4 })
        }
        .alignItems(HorizontalAlign.Start)
        Blank()
        Text(this.loggedIn ? this.nickname : '未登录')
          .fontSize(14)
          .fontColor('#5965F2')
      }
      .width('100%')
      .padding({ left: 24, right: 24, top: 28, bottom: 22 })

      if (!this.loggedIn) {
        this.LoginPanel()
      } else {
        this.HomePanel()
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor('#F6F7FB')
  }

  @Builder
  LoginPanel() {
    Column() {
      Text('登录 DateChat')
        .fontSize(22)
        .fontWeight(FontWeight.Medium)
        .width('100%')
        .margin({ bottom: 22 })

      TextInput({ text: this.serverUrl, placeholder: '服务端地址' })
        .width('100%')
        .height(52)
        .backgroundColor('#F2F3F8')
        .borderRadius(14)
        .padding({ left: 16, right: 16 })
        .onChange((value: string) => this.serverUrl = value.trim())

      TextInput({ text: this.phone, placeholder: '手机号' })
        .type(InputType.PhoneNumber)
        .width('100%')
        .height(52)
        .backgroundColor('#F2F3F8')
        .borderRadius(14)
        .padding({ left: 16, right: 16 })
        .margin({ top: 14 })
        .onChange((value: string) => this.phone = value.trim())

      Row() {
        TextInput({ text: this.code, placeholder: '验证码' })
          .type(InputType.Number)
          .layoutWeight(1)
          .height(52)
          .backgroundColor('#F2F3F8')
          .borderRadius(14)
          .padding({ left: 16, right: 16 })
          .onChange((value: string) => this.code = value.trim())
        Button('获取验证码')
          .height(52)
          .fontSize(14)
          .backgroundColor('#E8EAFF')
          .fontColor('#5965F2')
          .margin({ left: 10 })
          .enabled(!this.busy && this.phone.length >= 11)
          .onClick(() => this.sendCode())
      }
      .width('100%')
      .margin({ top: 14 })

      Button(this.busy ? '请稍候…' : '登录')
        .width('100%')
        .height(52)
        .fontSize(17)
        .fontWeight(FontWeight.Medium)
        .backgroundColor('#5965F2')
        .borderRadius(14)
        .margin({ top: 24 })
        .enabled(!this.busy && this.phone.length >= 11 && this.code.length > 0)
        .onClick(() => this.login())

      Text(this.serviceStatus)
        .fontSize(13)
        .fontColor('#858A96')
        .margin({ top: 18 })
    }
    .width('calc(100% - 32vp)')
    .padding(24)
    .backgroundColor(Color.White)
    .borderRadius(22)
  }

  @Builder
  HomePanel() {
    Column() {
      Text('你好，' + this.nickname)
        .fontSize(24)
        .fontWeight(FontWeight.Bold)
        .width('100%')
      Text('腾讯云 Chat 与 TRTC 原生鸿蒙 SDK 已就绪')
        .fontSize(14)
        .fontColor('#6A6F7A')
        .width('100%')
        .margin({ top: 8, bottom: 24 })

      Grid() {
        GridItem() { this.FeatureCard('消息', '即时聊天与系统通知', '#EEF0FF') }
        GridItem() { this.FeatureCard('附近', '发现同城用户', '#EAF9F2') }
        GridItem() { this.FeatureCard('通话', '语音与视频 TRTC', '#FFF2EA') }
        GridItem() { this.FeatureCard('我的', '账户与服务设置', '#F3EEFF') }
      }
      .columnsTemplate('1fr 1fr')
      .columnsGap(12)
      .rowsGap(12)
      .width('100%')
    }
    .width('calc(100% - 32vp)')
    .padding(24)
    .backgroundColor(Color.White)
    .borderRadius(22)
  }

  @Builder
  FeatureCard(title: string, subtitle: string, color: ResourceColor) {
    Column() {
      Text(title).fontSize(19).fontWeight(FontWeight.Medium)
      Text(subtitle).fontSize(12).fontColor('#747986').margin({ top: 8 })
    }
    .width('100%')
    .height(112)
    .padding(16)
    .alignItems(HorizontalAlign.Start)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(color)
    .borderRadius(16)
  }

  private normalizedBaseUrl(): string {
    return this.serverUrl.endsWith('/')
      ? this.serverUrl.substring(0, this.serverUrl.length - 1)
      : this.serverUrl;
  }

  private async sendCode(): Promise<void> {
    this.busy = true;
    try {
      const response: AxiosResponse<ApiEnvelope<SmsCodeData>> =
        await axios.post<ApiEnvelope<SmsCodeData>>(
          this.normalizedBaseUrl() + '/v1/auth/sms/code',
          { phone: this.phone, scene: 'login' }
        );
      if (response.data.code !== 0) {
        throw new Error(response.data.msg);
      }
      const debugCode: string = response.data.data.debug_code ?? '';
      if (debugCode.length > 0) {
        this.code = debugCode;
        this.serviceStatus = '开发环境验证码：' + debugCode;
      } else {
        this.serviceStatus = '验证码已发送';
      }
    } catch (error) {
      this.showError(error);
    } finally {
      this.busy = false;
    }
  }

  private async login(): Promise<void> {
    this.busy = true;
    try {
      const response: AxiosResponse<ApiEnvelope<LoginData>> =
        await axios.post<ApiEnvelope<LoginData>>(
          this.normalizedBaseUrl() + '/v1/auth/login',
          {
            phone: this.phone,
            code: this.code,
            device_id: 'harmonyos-device',
            platform: 3,
            app_version: '0.1.0-harmony'
          }
        );
      if (response.data.code !== 0) {
        throw new Error(response.data.msg);
      }
      this.nickname = response.data.data.user.nickname.length > 0
        ? response.data.data.user.nickname
        : '用户 ' + response.data.data.uid.toString();
      this.loggedIn = true;
      this.serviceStatus = response.data.data.im.enabled
        ? 'IM 登录凭据已由服务端安全下发'
        : '服务端尚未启用 IM';
    } catch (error) {
      this.showError(error);
    } finally {
      this.busy = false;
    }
  }

  private showError(error: Object): void {
    let message: string = '网络请求失败';
    if (error instanceof AxiosError) {
      message = error.message;
    } else if (error instanceof Error) {
      message = error.message;
    }
    this.serviceStatus = message;
    promptAction.showToast({ message: message, duration: 2500 });
  }
}
