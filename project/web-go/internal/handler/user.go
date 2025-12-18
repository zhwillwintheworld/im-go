package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"sudooom.im.web/internal/middleware"
	"sudooom.im.web/internal/repository"
	"sudooom.im.web/internal/service"
	"sudooom.im.web/pkg/response"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile 获取当前用户信息
// @Summary      获取当前用户信息
// @Description  获取当前登录用户的详细信息
// @Tags         用户
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=object{id=int64,object_code=string,username=string,nickname=string,avatar=string,status=int,create_at=time.Time}}
// @Failure      200  {object}  response.Response
// @Router       /user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{
		"id":          user.ID,
		"object_code": user.ObjectCode,
		"username":    user.Username,
		"nickname":    user.Nickname,
		"avatar":      user.Avatar,
		"status":      user.Status,
		"create_at":   user.CreateAt,
	})
}

// UpdateProfile 更新用户信息
// @Summary      更新用户信息
// @Description  更新当前登录用户的信息
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.UpdateProfileRequest true "更新信息"
// @Success      200  {object}  response.Response
// @Failure      200  {object}  response.Response
// @Router       /user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, err.Error())
		return
	}

	if err := h.userService.UpdateProfile(c.Request.Context(), userID, &req); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, nil)
}

// GetUserByID 获取指定用户信息
// @Summary      获取指定用户信息
// @Description  通过用户 ID 获取用户信息
// @Tags         用户
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "用户 ID"
// @Success      200  {object}  response.Response{data=object{id=int64,username=string,nickname=string,avatar=string}}
// @Failure      200  {object}  response.Response
// @Router       /user/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidParams, "invalid user id")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
	})
}

// Search 搜索用户
// @Summary      搜索用户
// @Description  根据关键词搜索用户
// @Tags         用户
// @Produce      json
// @Security     BearerAuth
// @Param        keyword query string true "搜索关键词"
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(20)
// @Success      200  {object}  response.Response{data=object{list=[]object{id=int64,username=string,nickname=string,avatar=string},page=int}}
// @Failure      200  {object}  response.Response
// @Router       /user/search [get]
func (h *UserHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.ErrorWithMsg(c, response.CodeInvalidParams, "keyword is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, err := h.userService.Search(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	var result []gin.H
	for _, u := range users {
		result = append(result, gin.H{
			"id":       u.ID,
			"username": u.Username,
			"nickname": u.Nickname,
			"avatar":   u.Avatar,
		})
	}

	response.Success(c, gin.H{
		"list": result,
		"page": page,
	})
}
