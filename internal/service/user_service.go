package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/security"
	"Cornerstone/internal/repository"
	"context"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
)

type UserService interface {
	Register(ctx context.Context, dto *dto.RegisterDTO) error
	Login(ctx context.Context, dto *dto.CredentialDTO, isPhoneCode bool) (string, error)
	Logout(ctx context.Context, token string) error
	GetUserInfo(ctx context.Context, id uint64) (*dto.UserDTO, error)
	GetUserHomeInfoById(ctx context.Context, id uint64) (*dto.UserDTO, error)
	GetUserSimpleInfoByIds(ctx context.Context, ids []uint64) ([]*dto.UserDTO, error)
	SearchUser(ctx context.Context, dto *dto.SearchUserDTO) ([]*model.User, error)
	UpdateUserInfo(ctx context.Context, id uint64, dto *dto.UserDTO) error
	UpdatePasswordFromToken(ctx context.Context, dto *dto.ForgetPasswordDTO) error
	UpdatePasswordFromOld(ctx context.Context, id uint64, dto *dto.ChangePasswordDTO) error
	UpdatePhone(ctx context.Context, id uint64, dto *dto.ChangePhoneDTO) error
	UpdateUsername(ctx context.Context, id uint64, dto *dto.ChangeUsernameDTO) error
	UpdateAvatar(ctx context.Context, id uint64, objectName string) error
	BanUser(ctx context.Context, id uint64) error
	UnBanUser(ctx context.Context, id uint64) error
	CancelUser(ctx context.Context, id uint64) error
}

type UserServiceImpl struct {
	userRepo repository.UserRepo
	roleRepo repository.RoleRepo
}

func NewUserService(userRepo repository.UserRepo, roleRepo repository.RoleRepo) UserService {
	return &UserServiceImpl{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (s *UserServiceImpl) Register(ctx context.Context, regDTO *dto.RegisterDTO) error {
	credentialDTO := &dto.CredentialDTO{
		Username: regDTO.Username,
		Phone:    regDTO.Phone,
	}
	findUser, err := s.findUserByLoginCredentials(ctx, credentialDTO)
	if err != nil {
		return err
	}
	if findUser != nil {
		return ErrUserUsernameExist
	}

	user := &model.User{}
	err = copier.Copy(user, &regDTO)
	if err != nil {
		return err
	}

	// username & password 形式注册
	if regDTO.Password != nil {
		passwordHash, err := security.HashPassword(*regDTO.Password)
		if err != nil {
			return err
		}
		user.Password = &passwordHash
	}

	// 手机号形式注册
	if regDTO.Phone != nil {
		key := consts.SmsCheckTokenKey + *regDTO.Phone
		value, err := redis.GetValue(ctx, key)
		if err != nil {
			return err
		}
		if value != *regDTO.PhoneToken {
			return ErrSmsRegTokenIncorrect
		}
		_ = redis.DeleteKey(ctx, key)
	}

	detail := &model.UserDetail{}
	err = copier.Copy(detail, &regDTO)
	if err != nil {
		return err
	}

	role := model.UserRole{
		UserID: user.ID,
		RoleID: 1,
	}
	roles := []*model.UserRole{&role}

	err = s.userRepo.CreateUser(ctx, user, detail, &roles)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserServiceImpl) Login(ctx context.Context, dto *dto.CredentialDTO, isByPassword bool) (string, error) {
	user, err := s.findUserByLoginCredentials(ctx, dto)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	if isByPassword {
		if dto.Password == nil || user.Password == nil {
			return "", ErrPasswordIncorrect
		}
		err = security.CheckPasswordHash(*dto.Password, *user.Password)
		if err != nil {
			return "", ErrPasswordIncorrect
		}
	}
	roleNames, err := s.getRoleNamesForUser(ctx, user)
	if err != nil {
		return "", err
	}
	token, err := security.GenerateToken(user.ID, roleNames)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *UserServiceImpl) Logout(ctx context.Context, token string) error {
	signature, err := security.ExtractSignature(token)
	if err != nil {
		return err
	}
	err = redis.SetWithExpiration(ctx, signature, true, time.Hour*24)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserServiceImpl) GetUserInfo(ctx context.Context, id uint64) (*dto.UserDTO, error) {
	user, err := s.userRepo.GetUserById(ctx, id)
	userDTO := &dto.UserDTO{}
	err = copier.Copy(userDTO, user)
	if err != nil {
		return nil, err
	}
	err = copier.Copy(userDTO, user.UserDetail)
	if err != nil {
		return nil, err
	}
	url := minio.GetPublicURL(user.UserDetail.AvatarURL)
	userDTO.AvatarURL = &url
	return userDTO, nil
}

func (s *UserServiceImpl) GetUserHomeInfoById(ctx context.Context, id uint64) (*dto.UserDTO, error) {
	key := consts.UserHomeInfoKey + strconv.FormatUint(id, 10)
	value, err := redis.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	if value != "" {
		var userDTO *dto.UserDTO
		err = json.Unmarshal([]byte(value), &userDTO)
		if err != nil {
			return nil, err
		}
		return userDTO, nil
	}
	user, err := s.userRepo.GetUserHomeInfoById(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	userDTO := &dto.UserDTO{}
	err = copier.Copy(userDTO, user)
	if err != nil {
		return nil, err
	}
	url := minio.GetPublicURL(user.AvatarURL)
	userDTO.AvatarURL = &url
	jsonStr, err := json.Marshal(userDTO)
	if err != nil {
		return nil, err
	}
	err = redis.SetWithExpiration(ctx, key, string(jsonStr), time.Hour*1)
	if err != nil {
		return nil, err
	}
	return userDTO, nil
}

func (s *UserServiceImpl) GetUserSimpleInfoByIds(ctx context.Context, ids []uint64) ([]*dto.UserDTO, error) {
	newIds := make([]uint64, 0, len(ids))
	mp := make(map[uint64]*dto.UserDTO)
	for _, id := range ids {
		value, err := redis.GetValue(ctx, consts.UserSimpleInfoKey+strconv.FormatUint(id, 10))
		if err != nil {
			return nil, err
		}
		if value != "" {
			var userDTO *dto.UserDTO
			err = json.Unmarshal([]byte(value), &userDTO)
			if err != nil {
				newIds = append(newIds, id)
			} else {
				mp[id] = userDTO
			}
		} else {
			newIds = append(newIds, id)
		}
	}
	if len(newIds) > 0 {
		userDetails, err := s.userRepo.GetUserSimpleInfoByIds(ctx, newIds)
		if err != nil {
			return nil, err
		}
		for _, userDetail := range userDetails {
			userDTO := &dto.UserDTO{}
			err = copier.Copy(userDTO, userDetail)
			if err != nil {
				return nil, err
			}
			url := minio.GetPublicURL(userDetail.AvatarURL)
			userDTO.AvatarURL = &url
			mp[userDetail.UserID] = userDTO
			jsonStr, err := json.Marshal(userDTO)
			if err != nil {
				return nil, err
			}
			err = redis.SetWithExpiration(ctx, consts.UserSimpleInfoKey+strconv.FormatUint(userDetail.UserID, 10), string(jsonStr), time.Hour*1)
			if err != nil {
				return nil, err
			}
		}
	}
	userDTOList := make([]*dto.UserDTO, 0, len(ids))
	for _, id := range ids {
		if mp[id] == nil {
			continue
		}
		userDTOList = append(userDTOList, mp[id])
	}
	return userDTOList, nil
}

func (s *UserServiceImpl) SearchUser(ctx context.Context, dto *dto.SearchUserDTO) ([]*model.User, error) {
	var user *model.User
	var userList []*model.User
	var err error
	if dto.ID != nil {
		user, err = s.userRepo.GetUserById(ctx, *dto.ID)
	} else if dto.Username != nil {
		user, err = s.userRepo.GetUserByUsername(ctx, *dto.Username)
	} else if dto.Phone != nil {
		user, err = s.userRepo.GetUserByPhone(ctx, *dto.Phone)
	} else if dto.Nickname != nil {
		userList, err = s.userRepo.GetUserByNickname(ctx, *dto.Nickname)
	}
	if err != nil {
		return nil, err
	}
	if user != nil {
		user.Password = nil
		return []*model.User{user}, nil
	}
	for _, item := range userList {
		item.Password = nil
		url := minio.GetPublicURL(item.UserDetail.AvatarURL)
		item.UserDetail.AvatarURL = url
	}
	return userList, nil
}

func (s *UserServiceImpl) UpdateUserInfo(ctx context.Context, id uint64, dto *dto.UserDTO) error {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	lockKey := consts.UserDetailLock + strconv.FormatUint(id, 10)
	lock, err := redis.TryLock(ctx, lockKey, newUUID.String(), time.Second*5, 3)
	if err != nil {
		return err
	}
	if !lock {
		return UnExpectedError
	}
	defer redis.UnLock(ctx, lockKey, newUUID.String())

	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	err = copier.CopyWithOption(&user.UserDetail, dto, copier.Option{IgnoreEmpty: true})
	if err != nil {
		return err
	}
	err = s.userRepo.UpdateUserDetail(ctx, &user.UserDetail)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserServiceImpl) UpdatePasswordFromToken(ctx context.Context, dto *dto.ForgetPasswordDTO) error {
	user, err := s.userRepo.GetUserByPhone(ctx, *dto.Phone)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	phone := user.Phone
	if phone == nil {
		return ErrUserPhoneNotFound
	}
	key := consts.SmsKey + *phone
	value, err := redis.GetValue(ctx, key)
	if err != nil {
		return err
	}
	if value != *dto.SmsCode {
		return ErrCodeIncorrect
	}
	passwordHash, err := security.HashPassword(*dto.NewPassword)
	if err != nil {
		return err
	}
	user.Password = &passwordHash
	err = s.userRepo.UpdateUser(ctx, user)
	_ = redis.DeleteKey(ctx, key)
	return err
}

func (s *UserServiceImpl) UpdatePasswordFromOld(ctx context.Context, id uint64, dto *dto.ChangePasswordDTO) error {
	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	err = security.CheckPasswordHash(*dto.OldPassword, *user.Password)
	if err != nil {
		return ErrPasswordIncorrect
	}
	passwordHash, err := security.HashPassword(*dto.NewPassword)
	if err != nil {
		return err
	}
	user.Password = &passwordHash
	return s.userRepo.UpdateUser(ctx, user)
}

func (s *UserServiceImpl) UpdatePhone(ctx context.Context, id uint64, dto *dto.ChangePhoneDTO) error {
	userByPhone, err := s.userRepo.GetUserByPhone(ctx, *dto.NewPhone)
	if err != nil {
		return err
	}
	if userByPhone != nil {
		return ErrUserPhoneExist
	}
	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	err = s.checkPhoneSmsCode(ctx, user.Phone, *dto.Token)
	if err != nil {
		return err
	}
	user.Phone = dto.NewPhone
	return s.userRepo.UpdateUser(ctx, user)
}

func (s *UserServiceImpl) UpdateUsername(ctx context.Context, id uint64, dto *dto.ChangeUsernameDTO) error {
	userByUsername, err := s.userRepo.GetUserByUsername(ctx, *dto.Username)
	if err != nil {
		return err
	}
	if userByUsername != nil {
		return ErrUserUsernameExist
	}
	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	user.Username = dto.Username
	return s.userRepo.UpdateUser(ctx, user)
}

func (s *UserServiceImpl) UpdateAvatar(ctx context.Context, id uint64, objectName string) error {
	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	user.UserDetail.AvatarURL = objectName
	err = s.userRepo.UpdateUserDetail(ctx, &user.UserDetail)
	if err != nil {
		return err
	}
	_ = redis.DeleteKey(ctx, consts.UserHomeInfoKey+strconv.FormatUint(id, 10))
	_ = redis.DeleteKey(ctx, consts.UserSimpleInfoKey+strconv.FormatUint(id, 10))
	return nil
}

func (s *UserServiceImpl) BanUser(ctx context.Context, id uint64) error {
	return s.changeUserIsBanStatus(ctx, id, true)
}

func (s *UserServiceImpl) UnBanUser(ctx context.Context, id uint64) error {
	return s.changeUserIsBanStatus(ctx, id, false)
}

func (s *UserServiceImpl) CancelUser(ctx context.Context, id uint64) error {
	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	return s.userRepo.DeleteUser(ctx, id)
}

func (s *UserServiceImpl) findUserByLoginCredentials(ctx context.Context, dto *dto.CredentialDTO) (*model.User, error) {
	if dto.Username != nil && *dto.Username != "" {
		return s.userRepo.GetUserByUsername(ctx, *dto.Username)
	}
	if dto.Phone != nil && *dto.Phone != "" {
		return s.userRepo.GetUserByPhone(ctx, *dto.Phone)
	}
	return nil, ErrMissingLoginCredentials
}

func (s *UserServiceImpl) getRoleNamesForUser(ctx context.Context, user *model.User) ([]string, error) {
	if len(user.UserRoles) == 0 {
		return []string{}, nil
	}
	roleIDs := make([]uint64, 0, len(user.UserRoles))
	for _, role := range user.UserRoles {
		roleIDs = append(roleIDs, role.RoleID)
	}
	roles, err := s.roleRepo.GetRoleByIDs(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	if roles == nil {
		return nil, UnExpectedError
	}
	roleNames := make([]string, 0, len(*roles))
	for _, role := range *roles {
		roleNames = append(roleNames, role.Name)
	}
	return roleNames, nil
}

func (s *UserServiceImpl) checkPhoneSmsCode(ctx context.Context, phone *string, code string) error {
	if phone == nil {
		return ErrUserPhoneNotFound
	}
	key := consts.SmsKey + *phone
	value, err := redis.GetValue(ctx, key)
	if err != nil {
		return ErrCodeIncorrect
	}
	if value != code {
		return ErrCodeIncorrect
	}
	_ = redis.DeleteKey(ctx, key)
	return nil
}

func (s *UserServiceImpl) changeUserIsBanStatus(ctx context.Context, id uint64, isBan bool) error {
	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	user.IsBan = isBan
	return s.userRepo.UpdateUser(ctx, user)
}
