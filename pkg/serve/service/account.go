package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"jank.com/jank_blog/internal/global"
	model "jank.com/jank_blog/internal/model/account"
	"jank.com/jank_blog/internal/utils"
	"jank.com/jank_blog/pkg/serve/controller/account/dto"
	"jank.com/jank_blog/pkg/serve/mapper"
)

var (
	registerLock      sync.Mutex // 用户注册锁， 保护并发用户注册的操作
	passwordResetLock sync.Mutex // 修改密码锁，保护并发修改用户密码的操作
	logoutLock        sync.Mutex // 用户登出锁，保护并发用户登出操作
)

const (
	AccAuthTokenCachePrefix         = "ACC_AUTH_TOKEN_CACHE_PREFIX"
	AccAuthTokenCacheExpire         = 60 * 15 // 15 分钟
	RefreshAuthTokenCachePrefix     = "REFRESH_AUTH_TOKEN_CACHE_PREFIX"
	RefreshAuthTokenCacheExpire     = 60 * 60 * 24 * 7 // 7 天
	RefreshAuthTokenCacheUserExpire = 60 * 3           // 3 分钟
)

// Register 用户注册逻辑
func RegisterUser(registerDto *dto.RegisterRequest, c echo.Context) (*model.Account, error) {
	registerLock.Lock()
	defer registerLock.Unlock()

	existingUser, _ := mapper.GetAccountByEmail(registerDto.Email)
	if existingUser != nil {
		utils.BizLogger(c).Errorf("邮箱(%s)已被注册", registerDto.Email)
		return nil, fmt.Errorf("邮箱已被注册")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerDto.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.BizLogger(c).Errorf("密码加密失败: %v", err)
		return nil, fmt.Errorf("密码加密失败")
	}

	acc := &model.Account{
		Email:    registerDto.Email,
		Password: string(hashedPassword),
		Nickname: registerDto.Nickname,
		Phone:    registerDto.Phone,
		RoleCode: "user",
	}

	if err := mapper.CreateAccount(acc); err != nil {
		return nil, err
	}

	go func() {
		global.BizLog.Infof("用户注册成功: %s\n", acc.Email)
	}()

	return acc, nil
}

// LoginUser 登录用户逻辑
func LoginUser(email, password, imgVerificationCode string, c echo.Context) (*dto.LoginResponse, error) {
	user, err := mapper.GetAccountByEmail(email)
	if err != nil {
		utils.BizLogger(c).Errorf("用户不存在: %v", err)
		return nil, fmt.Errorf("用户不存在")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		utils.BizLogger(c).Errorf("密码错误: %v", err)
		return nil, fmt.Errorf("密码错误")
	}

	accessTokenString, refreshTokenString, err := utils.GenerateJWT(uint(user.ID), user.Email)
	if err != nil {
		utils.BizLogger(c).Errorf("生成 token 失败: %v", err)
		return nil, fmt.Errorf("生成 token 失败: %v", err)
	}

	response := &dto.LoginResponse{
		UserId:       uint(user.ID),
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}

	return response, nil
}

// RefreshToken 刷新 token 逻辑
func LogoutUser(userId int64, c echo.Context) error {
	logoutLock.Lock()
	defer logoutLock.Unlock()

	// 生成用户鉴 token 和刷新 token 在 Redis 中的缓存键
	accKey := AccAuthTokenCachePrefix + strconv.FormatInt(userId, 10)
	refreshKey := RefreshAuthTokenCachePrefix + strconv.FormatInt(userId, 10)

	ctx := context.Background()

	go func() {
		cmd := global.Redis.Do(ctx, global.DelCmd, accKey)
		if cmd.Err() != nil {
			utils.BizLogger(c).Errorf("删除鉴权 token 缓存失败: %v", cmd.Err())
		}
	}()

	go func() {
		cmd := global.Redis.Do(ctx, global.DelCmd, refreshKey)
		if cmd.Err() != nil {
			utils.BizLogger(c).Errorf("删除刷新 token 缓存失败: %v", cmd.Err())
		}
	}()

	return nil
}

// ResetPassword 重置密码逻辑
func ResetPassword(userId int64, req *dto.ResetPwdRequest, c echo.Context) error {
	passwordResetLock.Lock()
	defer passwordResetLock.Unlock()

	if req.NewPassword != req.AgainNewPassword {
		utils.BizLogger(c).Errorf("两次密码输入不一致")
		return fmt.Errorf("两次密码输入不一致")
	}

	acc, err := mapper.GetAccountByUserID(userId)
	if err != nil {
		return err
	}
	if acc == nil {
		utils.BizLogger(c).Errorf("此用户不存在")
		return fmt.Errorf("此用户不存在")
	}

	newPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.BizLogger(c).Errorf("密码加密失败")
		return fmt.Errorf("密码加密失败")
	}
	acc.Password = string(newPassword)

	if err := mapper.UpdateAccount(acc); err != nil {
		utils.BizLogger(c).Errorf("修改密码失败: %v", err)
		return err
	}

	go func() {
		global.BizLog.Infof("用户密码已重置: %s", acc.Email)
	}()

	return nil
}
