package delivery

import (
	"github.com/jackc/pgx"
	"github.com/labstack/echo"
	"main/internal/models"
	"main/internal/users"
	"main/internal/users/usecase"
	"net/http"
)

type UserDelivery struct {
	userUseLogic users.UserUseCase
}

func NewUserDelivery(userU usecase.UserUseCaseRealisation) UserDelivery {
	return UserDelivery{userUseLogic: userU}
}

func (UserD UserDelivery) SetupHandlers(server *echo.Echo) {
	server.POST("/api/user/:nickname/create", UserD.CreateUser)
	server.GET("/api/user/:nickname/profile", UserD.GetUser)
	server.POST("/api/user/:nickname/profile", UserD.UpdateUser)
	server.GET("/api/service/status", UserD.GetStatus)
	server.POST("/api/service/clear", UserD.Clear)
}

func (UserD UserDelivery) GetStatus(rwContext echo.Context) error {
	return rwContext.JSON(http.StatusOK, UserD.userUseLogic.GetServerStatus())
}

func (UserD UserDelivery) Clear(rwContext echo.Context) error {
	UserD.userUseLogic.Clear()
	return rwContext.NoContent(http.StatusOK)
}

func (UserD UserDelivery) CreateUser(rwContext echo.Context) error {
	nickname := rwContext.Param("nickname")

	newUserData := new(models.UserModel)

	rwContext.Bind(newUserData)

	newUserData.Nickname = nickname

	answer, err := UserD.userUseLogic.CreateUser(*newUserData)

	if err != nil {
		return rwContext.JSON(http.StatusConflict, answer)
	}

	return rwContext.JSON(http.StatusCreated, answer)

}

func (UserD UserDelivery) GetUser(rwContext echo.Context) error {

	nickname := rwContext.Param("nickname")

	userData, err := UserD.userUseLogic.GetUser(nickname)

	if err != nil {
		return rwContext.JSON(http.StatusNotFound, &models.Error{Message: "Can't find user by nickname: " + nickname})
	}

	return rwContext.JSON(http.StatusOK, userData)

}

func (UserD UserDelivery) UpdateUser(rwContext echo.Context) error {

	nickname := rwContext.Param("nickname")

	newUserData := new(models.UserModel)

	rwContext.Bind(newUserData)

	newUserData.Nickname = nickname

	answer, err := UserD.userUseLogic.UpdateUserData(*newUserData)

	if err == pgx.ErrNoRows {
		return rwContext.JSON(http.StatusNotFound, &models.Error{Message: "Can't find user by nickname: " + nickname})
	}

	if err != nil {
		return rwContext.JSON(http.StatusConflict, &models.Error{Message: "This email is already registered by user: " + nickname})
	}

	return rwContext.JSON(http.StatusOK, answer)
}
