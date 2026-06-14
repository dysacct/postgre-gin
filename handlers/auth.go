// Login godoc
// @Summary      用户登录
// @Description  使用用户名密码登录获取 JWT token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "登录信息"
// @Success      200 {object} models.Response
// @Failure      400 {object} models.Response
// @Failure      401 {object} models.Response
// @Router       /login [post]

package handlers

import (
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"gin-postgre-project/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, err := utils.GeneratoToken(user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成Token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登陆成功",
		"data": gin.H{
			"token":    token,
			"username": user.Username,
			"role":     user.Role,
			"expire":   "72小时",
		},
	})
}
