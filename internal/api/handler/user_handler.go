package handler

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/service"
	"errors"
	log "log/slog"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	userSvc      service.UserService
	userRolesSvc service.UserRolesService
	smsSvc       service.SmsService
}

func NewUserHandler(userSvc service.UserService, userRolesSvc service.UserRolesService, smsSvc service.SmsService) *UserHandler {
	return &UserHandler{
		userSvc:      userSvc,
		userRolesSvc: userRolesSvc,
		smsSvc:       smsSvc,
	}
}

func (s *UserHandler) Register(c *gin.Context) {
	var registerDTO dto.RegisterDTO
	err := c.ShouldBind(&registerDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err = util.ValidateRegDTO(&registerDTO); err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.Register(c.Request.Context(), &registerDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) SendSmsCode(c *gin.Context) {
	var req dto.PhoneDTO
	err := c.ShouldBind(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	phone := req.Phone
	if !util.ValidatePhone(phone) {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}
	err = s.smsSvc.SendSms(c.Request.Context(), phone)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) Login(c *gin.Context) {
	var loginDTO dto.CredentialDTO
	err := c.ShouldBind(&loginDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	if !util.ValidateLoginDTO(&loginDTO) {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}
	token, err := s.userSvc.Login(c.Request.Context(), &loginDTO, true)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, map[string]string{
		"token": token,
	})
}

func (s *UserHandler) LoginByPhone(c *gin.Context) {
	var req dto.PhoneLoginDTO
	err := c.ShouldBind(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	phone := req.Phone
	code := req.Code
	token, err := s.smsSvc.CheckCode(c.Request.Context(), phone, code)
	if err != nil {
		response.Error(c, err)
		return
	}
	loginDTO := dto.CredentialDTO{
		Phone: &phone,
	}
	loginToken, err := s.userSvc.Login(c.Request.Context(), &loginDTO, false)
	isReg := false
	if err != nil && !errors.Is(err, service.ErrUserNotFound) {
		response.Fail(c, response.InternalServerError, err.Error())
		return
	}
	if !errors.Is(err, service.ErrUserNotFound) {
		_ = s.smsSvc.DelSmsRegToken(c.Request.Context(), phone)
		token = loginToken
		isReg = true
	}
	response.Success(c, map[string]any{
		"token":  token,
		"is_reg": isReg,
	})
}

func (s *UserHandler) ChangeUsername(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var changeUsernameDTO dto.ChangeUsernameDTO
	err := c.ShouldBind(&changeUsernameDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = util.ValidateDTO(&changeUsernameDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.UpdateUsername(c.Request.Context(), userID, &changeUsernameDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var changePasswordDTO dto.ChangePasswordDTO
	err := c.ShouldBind(&changePasswordDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = util.ValidateDTO(&changePasswordDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.UpdatePasswordFromOld(c.Request.Context(), userID, &changePasswordDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) ForgetPassword(c *gin.Context) {
	var forgetPasswordDTO dto.ForgetPasswordDTO
	err := c.ShouldBind(&forgetPasswordDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = util.ValidateDTO(&forgetPasswordDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.UpdatePasswordFromToken(c.Request.Context(), &forgetPasswordDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) ChangePhone(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var changePhoneDTO dto.ChangePhoneDTO
	err := c.ShouldBind(&changePhoneDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = util.ValidateDTO(&changePhoneDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.UpdatePhone(c.Request.Context(), userID, &changePhoneDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) Logout(c *gin.Context) {
	token := c.Request.Header.Get("Authorization")
	token = strings.Replace(token, "Bearer ", "", 1)
	err := s.userSvc.Logout(c.Request.Context(), token)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) GetUserInfo(c *gin.Context) {
	userID := c.GetUint64("user_id")
	userDTO, err := s.userSvc.GetUserInfo(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, userDTO)
}

func (s *UserHandler) UpdateUserInfo(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var userDTO dto.UserDTO
	err := c.ShouldBind(&userDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	userDTO.UserID = nil
	userDTO.Phone = nil
	userDTO.AvatarURL = nil
	userDTO.CreatedAt = nil
	err = util.ValidateDTO(&userDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.UpdateUserInfo(c.Request.Context(), userID, &userDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) BanUser(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var banUserDTO dto.BanUserDTO
	err := c.ShouldBind(&banUserDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.BanUser(c.Request.Context(), banUserDTO.UserID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) UnbanUser(c *gin.Context) {
	var banUserDTO dto.BanUserDTO
	err := c.ShouldBind(&banUserDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.UnBanUser(c.Request.Context(), banUserDTO.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) CancelUser(c *gin.Context) {
	userID := c.GetUint64("user_id")
	token := c.Request.Header.Get("Authorization")
	token = strings.Replace(token, "Bearer ", "", 1)
	err := s.userSvc.CancelUser(c.Request.Context(), userID, token)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) GetUserByCondition(c *gin.Context) {
	var conditionDTO dto.GetUserByConditionDTO
	err := c.ShouldBindQuery(&conditionDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	if conditionDTO.Page == 0 {
		conditionDTO.Page = 1
	}
	if conditionDTO.PageSize == 0 {
		conditionDTO.PageSize = 20
	}
	if conditionDTO.ID == nil && conditionDTO.Phone == nil && conditionDTO.Username == nil && conditionDTO.Nickname == nil {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}
	users, err := s.userSvc.GetUserByCondition(c.Request.Context(), &conditionDTO)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, users)
}

func (s *UserHandler) GetAllRoles(c *gin.Context) {
	roles, err := s.userRolesSvc.GetRoles(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, roles)
}

func (s *UserHandler) AddUserRole(c *gin.Context) {
	var userRole model.UserRole
	err := c.ShouldBind(&userRole)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userRolesSvc.AddRoleToUser(c.Request.Context(), userRole.UserID, userRole.RoleID)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.InvalidateUser(c.Request.Context(), userRole.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) DeleteUserRole(c *gin.Context) {
	var userRole model.UserRole
	err := c.ShouldBind(&userRole)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userRolesSvc.DeleteRoleFromUser(c.Request.Context(), userRole.UserID, userRole.RoleID)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userSvc.InvalidateUser(c.Request.Context(), userRole.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserHandler) GetHomeInfo(c *gin.Context) {
	query := c.Param("user_id")
	userID, err := strconv.ParseUint(query, 10, 64)
	user, err := s.userSvc.GetUserHomeInfoById(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, user)
}

func (s *UserHandler) GetUserSimpleInfoById(c *gin.Context) {
	query := c.Param("user_id")
	userID, err := strconv.ParseUint(query, 10, 64)
	user, err := s.userSvc.GetUserSimpleInfoByIds(c.Request.Context(), []uint64{userID})
	if err != nil {
		response.Error(c, err)
		return
	}
	if len(user) == 0 {
		response.Fail(c, response.NotFound, service.ErrUserNotFound.Error())
		return
	}
	response.Success(c, user[0])
}

func (s *UserHandler) GetUserSimpleInfoByIds(c *gin.Context) {
	query := c.Query("user_ids")
	if query == "" {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}
	query = query[1 : len(query)-1]
	userIDs := strings.Split(query, ",")
	userIDsUint64 := make([]uint64, 0, len(userIDs))
	for _, userID := range userIDs {
		userIDUint64, err := strconv.ParseUint(userID, 10, 64)
		if err != nil {
			response.Error(c, err)
			return
		}
		userIDsUint64 = append(userIDsUint64, userIDUint64)
	}
	userDTOList, err := s.userSvc.GetUserSimpleInfoByIds(c.Request.Context(), userIDsUint64)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, userDTOList)
}

func (s *UserHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetUint64("user_id")
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}

	reader, err := file.Open()
	if err != nil {
		response.Error(c, err)
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	contentType, err := util.GetSafeContentType(reader)
	if err != nil {
		response.Error(c, err)
		return
	}
	if !strings.HasPrefix(contentType, consts.MimePrefixImage) {
		response.Error(c, service.ErrFileNotSupported)
		return
	}

	ext := path.Ext(file.Filename)
	objectName := "avatars/" + uuid.NewString() + ext
	fileKey, err := minio.UploadFile(c.Request.Context(), objectName, reader, file.Size, contentType)
	if err != nil {
		log.ErrorContext(c, "MinIO upload failed", "err", err)
		response.Error(c, service.UnExpectedError)
		return
	}

	err = s.userSvc.UpdateAvatar(c.Request.Context(), userID, fileKey)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// SearchUser 普通用户搜人 (脱敏数据)
func (s *UserHandler) SearchUser(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if keyword == "" {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	users, err := s.userSvc.SearchUser(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, users)
}
