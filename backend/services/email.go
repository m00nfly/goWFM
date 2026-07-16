package services

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"goWFM/config"
)

const ResetPasswordTemplateKey = "reset_password"

// SMTPErrorDetails 从被多层包装的 SMTP 错误中提取服务器返回的状态码与原始信息。
func SMTPErrorDetails(err error) (code int, message string, ok bool) {
	var smtpErr *textproto.Error
	if !errors.As(err, &smtpErr) {
		return 0, "", false
	}
	return smtpErr.Code, strings.TrimSpace(smtpErr.Msg), true
}

type ResetPasswordTemplateData struct {
	SiteName       string
	Username       string
	ResetURL       string
	ExpiresMinutes int
}

// EffectiveSenderName 返回实际使用的发件人名称：显式配置、站点名称、goWFM。
func EffectiveSenderName(configuredName string) string {
	if name := strings.TrimSpace(configuredName); name != "" {
		return name
	}
	if siteName := strings.TrimSpace(config.GetBasic().SiteName); siteName != "" {
		return siteName
	}
	return "goWFM"
}

// ValidateEmailSettings 校验 SMTP、发件人与当前已支持的邮件模板。
func ValidateEmailSettings(cfg config.EmailSettings) error {
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		return fmt.Errorf("SMTP 服务器不能为空")
	}
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return fmt.Errorf("SMTP 端口无效")
	}
	if strings.ContainsAny(cfg.SenderName, "\r\n") {
		return fmt.Errorf("发件人名称不能包含换行符")
	}
	if err := validateMailbox(cfg.SenderEmail); err != nil {
		return fmt.Errorf("发件人 Email 无效: %w", err)
	}
	tpl, ok := cfg.Templates[ResetPasswordTemplateKey]
	if !ok {
		return fmt.Errorf("缺少重置密码邮件模板")
	}
	_, _, err := RenderResetPasswordEmail(tpl, ResetPasswordTemplateData{
		SiteName: "goWFM", Username: "example", ResetURL: "https://example.com/login?reset_token=preview", ExpiresMinutes: 15,
	})
	return err
}

func validateMailbox(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("邮箱不能为空")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return fmt.Errorf("邮箱格式不正确")
	}
	return nil
}

func RenderResetPasswordEmail(tpl config.EmailTemplate, data ResetPasswordTemplateData) (string, string, error) {
	if err := validateResetTemplateVariables(tpl.Subject + "\n" + tpl.HTML); err != nil {
		return "", "", err
	}
	subjectTpl, err := texttemplate.New("subject").Option("missingkey=error").Parse(tpl.Subject)
	if err != nil {
		return "", "", fmt.Errorf("邮件主题模板无效: %w", err)
	}
	var subject bytes.Buffer
	if err := subjectTpl.Execute(&subject, data); err != nil {
		return "", "", fmt.Errorf("邮件主题模板渲染失败: %w", err)
	}
	if strings.TrimSpace(subject.String()) == "" || strings.ContainsAny(subject.String(), "\r\n") {
		return "", "", fmt.Errorf("邮件主题不能为空或包含换行符")
	}

	bodyTpl, err := template.New("html").Option("missingkey=error").Parse(tpl.HTML)
	if err != nil {
		return "", "", fmt.Errorf("HTML 模板无效: %w", err)
	}
	var body bytes.Buffer
	if err := bodyTpl.Execute(&body, data); err != nil {
		return "", "", fmt.Errorf("HTML 模板渲染失败: %w", err)
	}
	if strings.TrimSpace(body.String()) == "" {
		return "", "", fmt.Errorf("HTML 模板不能为空")
	}
	return subject.String(), body.String(), nil
}

var templateActionPattern = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)

func validateResetTemplateVariables(value string) error {
	allowed := map[string]bool{".SiteName": true, ".Username": true, ".ResetURL": true, ".ExpiresMinutes": true}
	for _, match := range templateActionPattern.FindAllStringSubmatch(value, -1) {
		action := strings.TrimSpace(match[1])
		if !allowed[action] {
			return fmt.Errorf("不支持的模板变量或表达式 {{%s}}", action)
		}
	}
	return nil
}

func SendResetPasswordEmail(to, username, token string, expiresMinutes int) error {
	cfg := config.GetEmail()
	tpl, ok := cfg.Templates[ResetPasswordTemplateKey]
	if !ok {
		tpl = config.DefaultResetPasswordTemplate()
	}
	basic := config.GetBasic()
	baseURL := strings.TrimRight(strings.TrimSpace(basic.SiteLink), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("系统设置中的站点链接必须是完整的 HTTP(S) 地址")
	}
	siteName := strings.TrimSpace(basic.SiteName)
	if siteName == "" {
		siteName = "goWFM"
	}
	resetURL := baseURL + "/login?reset_token=" + url.QueryEscape(token)
	subject, body, err := RenderResetPasswordEmail(tpl, ResetPasswordTemplateData{
		SiteName: siteName, Username: username, ResetURL: resetURL, ExpiresMinutes: expiresMinutes,
	})
	if err != nil {
		return err
	}
	return SendHTMLMail(cfg, to, subject, body)
}

func SendTestEmail(to string) error {
	siteName := strings.TrimSpace(config.GetBasic().SiteName)
	if siteName == "" {
		siteName = "goWFM"
	}
	body := `<!doctype html><html lang="zh-CN"><body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;padding:24px;color:#172033"><h2>邮件设置测试成功</h2><p>` + template.HTMLEscapeString(siteName) + ` 已成功连接 SMTP 服务并发送此邮件。</p></body></html>`
	return SendHTMLMail(config.GetEmail(), to, "["+siteName+"] 邮件发送测试", body)
}

func SendHTMLMail(cfg config.EmailSettings, to, subject, htmlBody string) error {
	cfg.SenderName = EffectiveSenderName(cfg.SenderName)
	if err := ValidateEmailSettings(cfg); err != nil {
		return err
	}
	if err := validateMailbox(to); err != nil {
		return fmt.Errorf("收件人 Email 无效: %w", err)
	}
	from := mail.Address{Name: strings.TrimSpace(cfg.SenderName), Address: strings.TrimSpace(cfg.SenderEmail)}
	toAddress := mail.Address{Address: strings.TrimSpace(to)}
	domain := cfg.SenderEmail[strings.LastIndex(cfg.SenderEmail, "@")+1:]
	messageID := fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), strconv.FormatInt(time.Now().Unix(), 36), domain)
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from.String())
	fmt.Fprintf(&msg, "To: %s\r\n", toAddress.String())
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&msg, "Message-ID: %s\r\n", messageID)
	fmt.Fprint(&msg, "MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	msg.WriteString(htmlBody)

	host := strings.TrimSpace(cfg.SMTPHost)
	address := net.JoinHostPort(host, strconv.Itoa(cfg.SMTPPort))
	client, err := newSMTPClient(address, host, cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	if cfg.SMTPUsername != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, host)); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("SMTP 发件人被拒绝: %w", err)
	}
	if err := client.Rcpt(toAddress.Address); err != nil {
		return fmt.Errorf("SMTP 收件人被拒绝: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP 无法写入邮件: %w", err)
	}
	if _, err = io.Copy(w, &msg); err != nil {
		w.Close()
		return fmt.Errorf("SMTP 写入邮件失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP 发送邮件失败: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("SMTP 结束会话失败: %w", err)
	}
	return nil
}

func newSMTPClient(address, host string, cfg config.EmailSettings) (*smtp.Client, error) {
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, InsecureSkipVerify: cfg.SkipTLSVerify} //nolint:gosec -- controlled admin option for private SMTP servers
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if cfg.EnableTLS && cfg.SMTPPort == 465 {
		conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("连接 SMTP TLS 服务失败: %w", err)
		}
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("创建 SMTP 客户端失败: %w", err)
		}
		return client, nil
	}
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("连接 SMTP 服务失败: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	if cfg.EnableTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			client.Close()
			return nil, fmt.Errorf("SMTP 服务器不支持 STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			client.Close()
			return nil, fmt.Errorf("SMTP STARTTLS 失败: %w", err)
		}
	}
	return client, nil
}
