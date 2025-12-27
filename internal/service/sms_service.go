package service

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type SmsService interface {
	SendSms(ctx context.Context, phone string) error
	CheckCode(ctx context.Context, phone string, code string) (string, error)
	DelSmsRegToken(ctx context.Context, phone string) error
}

type SmsServiceImpl struct{}

func NewSmsService() SmsService {
	return &SmsServiceImpl{}
}

func (s *SmsServiceImpl) SendSms(ctx context.Context, phone string) error {
	code := util.GenerateCode(6)
	err := redis.SetWithExpiration(ctx, consts.SmsKey+phone, code, 10*time.Minute)
	if err != nil {
		return err
	}
	err = util.SendSms(phone, code)
	if err != nil {
		return err
	}
	return nil
}

func (s *SmsServiceImpl) CheckCode(ctx context.Context, phone string, code string) (string, error) {
	redisCode, err := redis.GetValue(ctx, consts.SmsKey+phone)
	if err != nil {
		return "", err
	}
	if redisCode != code {
		return "", ErrCodeIncorrect
	}
	_ = redis.DeleteKey(ctx, consts.SmsKey+phone)
	tempToken := strconv.Itoa(int(uuid.New().ID()))
	err = redis.SetWithExpiration(ctx, consts.SmsCheckTokenKey+phone, tempToken, 1*time.Hour)
	if err != nil {
		return "", err
	}
	return tempToken, nil
}

func (s *SmsServiceImpl) DelSmsRegToken(ctx context.Context, phone string) error {
	return redis.DeleteKey(ctx, consts.SmsCheckTokenKey+phone)
}
