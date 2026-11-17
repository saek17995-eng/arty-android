// Package sms 抽象短信验证码下发通道。
//
// 两个实现：
//   - Log：只写日志不发短信，本地开发与自动化测试用；
//   - Tencent：腾讯云短信（SendSms，API 2021-01-11）。
//
// 业务侧只依赖 Provider 接口，切换通道改配置即可，不用动代码。
package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/datechat/server/internal/pkg/tc3"
)

// Message 是一条待发送的验证码短信。
type Message struct {
	Phone string        // 11 位手机号，不含 +86
	Code  string        // 验证码
	TTL   time.Duration // 有效期，用于模板里的"X 分钟内有效"
}

// Provider 短信通道。
// 返回 requestID 便于在服务商控制台对账；发送失败必须返回 error，由上层决定是否重试。
type Provider interface {
	Name() string
	Send(ctx context.Context, msg Message) (requestID string, err error)
}

// —— 本地日志通道 ——

// LogProvider 把验证码打进日志，不产生真实短信费用。
type LogProvider struct {
	log *zap.Logger
}

// NewLogProvider 创建日志通道。
func NewLogProvider(log *zap.Logger) *LogProvider {
	return &LogProvider{log: log}
}

// Name 实现 Provider。
func (p *LogProvider) Name() string { return "log" }

// Send 实现 Provider。
func (p *LogProvider) Send(_ context.Context, msg Message) (string, error) {
	if p.log != nil {
		p.log.Info("【本地短信通道】验证码未真实发送",
			zap.String("phone", maskPhone(msg.Phone)),
			zap.String("code", msg.Code),
			zap.Duration("ttl", msg.TTL),
		)
	}
	return "local", nil
}

// —— 腾讯云短信通道 ——

// TencentConfig 腾讯云短信配置。
type TencentConfig struct {
	Cred       tc3.Credential
	SdkAppID   string
	SignName   string
	TemplateID string
	Region     string
	Endpoint   string
}

// TencentProvider 腾讯云短信。
type TencentProvider struct {
	cfg    TencentConfig
	client *tc3.Client
}

// NewTencentProvider 创建腾讯云短信通道。
func NewTencentProvider(cfg TencentConfig, timeout time.Duration) (*TencentProvider, error) {
	if !cfg.Cred.Valid() {
		return nil, errors.New("sms: 腾讯云短信缺少 secret_id / secret_key")
	}
	if cfg.SdkAppID == "" || cfg.SignName == "" || cfg.TemplateID == "" {
		return nil, errors.New("sms: 腾讯云短信缺少 sdk_app_id / sign_name / template_id")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "sms.tencentcloudapi.com"
	}
	return &TencentProvider{cfg: cfg, client: tc3.New(cfg.Cred, "sms", cfg.Endpoint, timeout)}, nil
}

// Name 实现 Provider。
func (p *TencentProvider) Name() string { return "tencent" }

// sendStatus 是 SendSms 返回的单条发送状态。
type sendStatus struct {
	SerialNo  string `json:"SerialNo"`
	PhoneNum  string `json:"PhoneNumber"`
	Fee       int    `json:"Fee"`
	SessionNo string `json:"SessionContext"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
}

// Send 实现 Provider。
// 模板约定两个变量：{1} 验证码、{2} 有效分钟数，与 0002_seed 里的模板保持一致。
func (p *TencentProvider) Send(ctx context.Context, msg Message) (string, error) {
	minutes := int(msg.TTL.Minutes())
	if minutes <= 0 {
		minutes = 5
	}
	payload := map[string]any{
		"SmsSdkAppId":      p.cfg.SdkAppID,
		"SignName":         p.cfg.SignName,
		"TemplateId":       p.cfg.TemplateID,
		"PhoneNumberSet":   []string{"+86" + msg.Phone},
		"TemplateParamSet": []string{msg.Code, strconv.Itoa(minutes)},
	}

	raw, err := p.client.Call(ctx, "SendSms", "2021-01-11", p.cfg.Region, payload)
	if err != nil {
		return "", err
	}

	var resp struct {
		SendStatusSet []sendStatus `json:"SendStatusSet"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("sms: 解析 SendSms 响应失败: %w", err)
	}
	if len(resp.SendStatusSet) == 0 {
		return "", errors.New("sms: SendSms 未返回发送状态")
	}
	st := resp.SendStatusSet[0]
	if st.Code != "Ok" {
		return st.SerialNo, fmt.Errorf("sms: 发送失败 %s: %s", st.Code, st.Message)
	}
	return st.SerialNo, nil
}

// YunpianConfig 云片短信配置。
type YunpianConfig struct {
	APIKey          string
	TemplateID      int64
	CodeVariable    string
	MinutesVariable string
	Endpoint        string
}

// YunpianProvider 云片国内短信模板通道。
type YunpianProvider struct {
	cfg    YunpianConfig
	client *http.Client
}

// NewYunpianProvider 创建云片短信通道。
func NewYunpianProvider(cfg YunpianConfig, timeout time.Duration) (*YunpianProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("sms: 云片短信缺少 api_key")
	}
	if cfg.TemplateID <= 0 {
		return nil, errors.New("sms: 云片短信 template_id 必须为正数")
	}
	if strings.TrimSpace(cfg.CodeVariable) == "" {
		cfg.CodeVariable = "#code#"
	}
	if strings.TrimSpace(cfg.MinutesVariable) == "" {
		cfg.MinutesVariable = "#min#"
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		cfg.Endpoint = "https://sms.yunpian.com/v2/sms/tpl_single_send.json"
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &YunpianProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}, nil
}

// Name 实现 Provider。
func (p *YunpianProvider) Name() string { return "yunpian" }

// Send 使用云片指定模板单发接口。模板变量默认是 #code# 和 #min#。
func (p *YunpianProvider) Send(ctx context.Context, msg Message) (string, error) {
	minutes := int(msg.TTL.Minutes())
	if minutes <= 0 {
		minutes = 5
	}
	form := url.Values{
		"apikey":    {p.cfg.APIKey},
		"mobile":    {msg.Phone},
		"tpl_id":    {strconv.FormatInt(p.cfg.TemplateID, 10)},
		"tpl_value": {url.QueryEscape(p.cfg.CodeVariable) + "=" + url.QueryEscape(msg.Code) + "&" + url.QueryEscape(p.cfg.MinutesVariable) + "=" + url.QueryEscape(strconv.Itoa(minutes))},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("sms: 创建云片请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json;charset=utf-8")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sms: 云片请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("sms: 读取云片响应失败: %w", err)
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		SID  int64  `json:"sid"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("sms: 解析云片响应失败（HTTP %d）: %w", resp.StatusCode, err)
	}
	if resp.StatusCode/100 != 2 || result.Code != 0 {
		return strconv.FormatInt(result.SID, 10), fmt.Errorf("sms: 云片发送失败 %d: %s", result.Code, result.Msg)
	}
	return strconv.FormatInt(result.SID, 10), nil
}

// AliyunConfig 阿里云短信 RPC API 配置。
type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string
}

// AliyunProvider 阿里云 Dysmsapi 短信通道。
type AliyunProvider struct {
	cfg    AliyunConfig
	client *http.Client
}

func NewAliyunProvider(cfg AliyunConfig, timeout time.Duration) (*AliyunProvider, error) {
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, errors.New("sms: 阿里云短信缺少 access_key_id / access_key_secret")
	}
	if strings.TrimSpace(cfg.SignName) == "" || strings.TrimSpace(cfg.TemplateCode) == "" {
		return nil, errors.New("sms: 阿里云短信缺少 sign_name / template_code")
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		cfg.Endpoint = "https://dysmsapi.aliyuncs.com/"
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &AliyunProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}, nil
}

func (p *AliyunProvider) Name() string { return "aliyun" }

// Send 调用阿里云 SendSms RPC API。模板变量为 {"code":"验证码"}；有效期文案由已审核模板固定。
func (p *AliyunProvider) Send(ctx context.Context, msg Message) (string, error) {
	params := map[string]string{
		"AccessKeyId": p.cfg.AccessKeyID, "Action": "SendSms", "Format": "JSON",
		"PhoneNumbers": msg.Phone, "SignName": p.cfg.SignName, "SignatureMethod": "HMAC-SHA1",
		"SignatureNonce": uuidNonce(), "SignatureVersion": "1.0", "SignatureTimestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"TemplateCode": p.cfg.TemplateCode, "TemplateParam": `{"code":"` + msg.Code + `"}`, "Version": "2017-05-25",
	}
	canonical := canonicalQuery(params)
	stringToSign := "GET&%2F&" + percentEncode(percentEncode(canonical))
	h := hmac.New(sha1.New, []byte(p.cfg.AccessKeySecret+"&"))
	_, _ = h.Write([]byte(stringToSign))
	params["Signature"] = base64.StdEncoding.EncodeToString(h.Sum(nil))
	query := canonicalQuery(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.cfg.Endpoint, "?")+"?"+query, nil)
	if err != nil {
		return "", fmt.Errorf("sms: 创建阿里云请求失败: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sms: 阿里云请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var result struct{ Code, Message, RequestID, BizID string }
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("sms: 解析阿里云响应失败（HTTP %d）: %w", resp.StatusCode, err)
	}
	if resp.StatusCode/100 != 2 || result.Code != "OK" {
		return result.RequestID, fmt.Errorf("sms: 阿里云发送失败 %s: %s", result.Code, result.Message)
	}
	return result.BizID, nil
}

func uuidNonce() string { return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix()) }

func percentEncode(s string) string {
	return strings.NewReplacer("+", "%20", "*", "%2A", "%7E", "~").Replace(url.QueryEscape(s))
}

func canonicalQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, percentEncode(k)+"="+percentEncode(params[k]))
	}
	return strings.Join(parts, "&")
}

// maskPhone 日志脱敏，避免日志成为手机号泄露源。
func maskPhone(p string) string {
	if len(p) != 11 {
		return "***"
	}
	return p[:3] + "****" + p[7:]
}
