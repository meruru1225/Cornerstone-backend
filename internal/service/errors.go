package service

import (
	"errors"
)

const (
	BadRequest          = 400
	Unauthorized        = 401
	NotFound            = 404
	InternalServerError = 500
)

var (
	ErrParamInvalid            = errors.New("参数错误")
	ErrUserNotFound            = errors.New("用户不存在")
	ErrUserBan                 = errors.New("用户已被封禁")
	ErrUserBanSelf             = errors.New("不能封禁自己")
	ErrUserBanAdmin            = errors.New("不能封禁管理员")
	ErrUserExist               = errors.New("用户已存在")
	ErrUserPhoneNotFound       = errors.New("手机号未注册")
	ErrUserPhoneExist          = errors.New("手机号已注册")
	ErrUserUsernameExist       = errors.New("用户名已存在")
	ErrPasswordIncorrect       = errors.New("密码错误")
	ErrCodeIncorrect           = errors.New("验证码错误")
	ErrMissingLoginCredentials = errors.New("缺少登录凭据")
	ErrSmsRegTokenIncorrect    = errors.New("短信注册token错误")
	ErrFileNotSupported        = errors.New("不支持的文件类型")
	ErrFileNotExist            = errors.New("文件不存在")
	ErrUserFollowExist         = errors.New("用户已关注")
	ErrUserFollowLimit         = errors.New("用户关注数量超过限制")
	ErrUserFollowSelf          = errors.New("用户不能关注自己")
	ErrUserHasRole             = errors.New("用户已拥有此角色")
	ErrPostNotFound            = errors.New("帖子不存在")
	ErrPostCommentNotFound     = errors.New("评论不存在")
	ErrActionDuplicate         = errors.New("重复操作")
	ErrSysBoxNotFound          = errors.New("系统通知不存在")
	ErrTargetUserInvalid       = errors.New("目标用户无效")
	ErrConversation            = errors.New("会话异常")
	UnauthorizedError          = errors.New("权限不足")
	UnExpectedError            = errors.New("系统异常，请稍后重试")
)

var ErrorMap = map[error]int{
	ErrParamInvalid:            BadRequest,
	ErrUserNotFound:            NotFound,
	ErrUserBan:                 Unauthorized,
	ErrUserBanSelf:             Unauthorized,
	ErrUserBanAdmin:            Unauthorized,
	ErrUserExist:               BadRequest,
	ErrUserPhoneNotFound:       NotFound,
	ErrUserPhoneExist:          BadRequest,
	ErrUserUsernameExist:       BadRequest,
	ErrPasswordIncorrect:       Unauthorized,
	ErrCodeIncorrect:           Unauthorized,
	ErrMissingLoginCredentials: Unauthorized,
	ErrSmsRegTokenIncorrect:    Unauthorized,
	ErrFileNotSupported:        BadRequest,
	ErrFileNotExist:            NotFound,
	ErrUserFollowExist:         BadRequest,
	ErrUserFollowLimit:         BadRequest,
	ErrUserFollowSelf:          BadRequest,
	ErrUserHasRole:             BadRequest,
	ErrPostNotFound:            NotFound,
	ErrPostCommentNotFound:     NotFound,
	ErrActionDuplicate:         BadRequest,
	ErrSysBoxNotFound:          NotFound,
	ErrTargetUserInvalid:       BadRequest,
	ErrConversation:            BadRequest,
	UnauthorizedError:          Unauthorized,
	UnExpectedError:            InternalServerError,
}
