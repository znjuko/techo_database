package main

import (
	"fmt"
	"github.com/labstack/echo"
	fD "main/internal/forums/delivery"
	fR "main/internal/forums/repository"
	fU "main/internal/forums/usecase"
	pD "main/internal/posts/delivery"
	pR "main/internal/posts/repository"
	pU "main/internal/posts/usecase"
	"time"

	tD "main/internal/threads/delivery"
	tR "main/internal/threads/repository"
	tU "main/internal/threads/usecase"

	"github.com/jackc/pgx"
	"main/internal/users/delivery"
	"main/internal/users/repository"
	"main/internal/users/usecase"
)

const (
	usernameDB = "docker"
	passwordDB = "docker"
	nameDB     = "docker"
)

type RequestHandler struct {
	userHandler   delivery.UserDelivery
	forumHandler  fD.ForumDelivery
	threadHandler tD.ThreadDelivery
	postHandler   pD.PostDelivery
}

func StartServer(db *pgx.ConnPool) *RequestHandler {

	postDB := pR.NewPostRepoRealisation(db)
	postUse := pU.NewPostUseRealistaion(postDB)
	postH := pD.NewUserDelivery(postUse)

	threadDB := tR.NewThreadRepoRealisation(db)
	threadUse := tU.NewThreadsUseRealisation(threadDB)
	threadH := tD.NewThreadDeliveryRealisation(threadUse)
	forumDB := fR.NewForumRepoRealisation(db)
	forumUse := fU.NewForumUseCaseRealisation(forumDB)
	forumH := fD.NewForumDeliveryRealisation(forumUse)
	userDB := repository.NewUserRepoRealisation(db)
	userUse := usecase.NewUserUseCaseRealisation(userDB)
	userH := delivery.NewUserDelivery(userUse)

	api := &RequestHandler{userHandler: userH, forumHandler: forumH, threadHandler: threadH, postHandler: postH}

	return api
}

func JSONMiddleware(next echo.HandlerFunc) echo.HandlerFunc {

	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		return next(c)
	}
}

func Logs(next echo.HandlerFunc) echo.HandlerFunc {

	return func(rwContext echo.Context) error {

		start := time.Now()

		err := next(rwContext)

		respTime := time.Since(start)

		fmt.Println("MICRO SEC:" ,respTime.Microseconds(), "\n PATH:" ,rwContext.Request().URL.Path, "\n METHOD:" , rwContext.Request().Method)

		return err

	}
}

func main() {

	server := echo.New()

	server.Use(JSONMiddleware)
	server.Use(Logs)

	connectString := "user=" + usernameDB + " password=" + passwordDB + " dbname=" + nameDB + " sslmode=disable"

	pgxConn, err := pgx.ParseConnectionString(connectString)
	pgxConn.PreferSimpleProtocol = true
	if err != nil {
		server.Logger.Fatal("PARSING CONFIG ERROR", err.Error())
	}

	config := pgx.ConnPoolConfig{
		ConnConfig:     pgxConn,
		MaxConnections: 16,
		AfterConnect:   nil,
		AcquireTimeout: 0,
	}

	//config.Host = "localhost"
	//config.Port = 5432
	//config.Database = nameDB
	//config.User = usernameDB
	//config.Password = passwordDB

	connPool, err := pgx.NewConnPool(config)
	defer connPool.Close()
	if err != nil {
		server.Logger.Fatal("NO CONNECTION TO BD", err.Error())
	}
	fmt.Println(connPool.Stat())
	api := StartServer(connPool)
	api.userHandler.SetupHandlers(server)
	api.forumHandler.SetupHandlers(server)
	api.threadHandler.SetupHandlers(server)
	api.postHandler.SetupHandlers(server)

	server.Logger.Fatal(server.Start(":5000"))
}
