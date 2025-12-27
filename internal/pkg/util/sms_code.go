package util

import (
	"Cornerstone/internal/api/config"
	"fmt"
	"io"
	log "log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

const SuccessResp = "0"
const digits = "0123456789"

func SendSms(phone string, code string) error {
	smsCfg := config.Cfg.SMS
	content := url.QueryEscape(fmt.Sprintf("【CornerStone】您的验证码为 %s 。", code))
	fullUrl := fmt.Sprintf("%s?u=%s&p=%s&m=%s&c=%s", smsCfg.URL, smsCfg.Username, smsCfg.ApiKey, phone, content)

	log.Info(fmt.Sprintf("调用短信接口: %s", fullUrl))
	log.Info(fmt.Sprintf("发送给 %s 的验证码为 %s", phone, code))

	client := http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest(http.MethodGet, fullUrl, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("sms send failed: %s", response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if string(body) != SuccessResp {
		return fmt.Errorf("sms send failed: response code %s", string(body))
	}
	log.Info(fmt.Sprintf("短信接口响应: %s", string(body)))
	return nil
}

func GenerateCode(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := make([]byte, length)
	for i := range code {
		code[i] = digits[r.Intn(len(digits))]
	}
	return string(code)
}
