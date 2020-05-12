package repository

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx"
	_ "github.com/lib/pq"
	"main/internal/models"
	"strconv"
)

type UserRepoRealisation struct {
	database *pgx.ConnPool
}

func NewUserRepoRealisation(db *pgx.ConnPool) UserRepoRealisation {
	return UserRepoRealisation{database: db}
}

func (UserData UserRepoRealisation) CreateNewUser(userModel models.UserModel) ([]models.UserModel, error) {

	allData := make([]models.UserModel, 0)
	var err error

	_, err = UserData.database.Exec("INSERT INTO users (nickname , fullname , email , about) VALUES($1 , $2 , $3 ,$4)", userModel.Nickname, userModel.Fullname, userModel.Email, userModel.About)

	if err != nil {
		//fmt.Println(err)
		row, err := UserData.database.Query("SELECT nickname , fullname , email , about FROM users WHERE nickname = $1 OR email = $2", userModel.Nickname, userModel.Email)

		if row != nil {
			for row.Next() {

				if err == nil {
					err = errors.New("such user already exists")
				}

				existingUser := models.UserModel{
					Nickname: "",
					Fullname: "",
					Email:    "",
					About:    "",
				}

				row.Scan(&existingUser.Nickname, &existingUser.Fullname, &existingUser.Email, &existingUser.About)

				allData = append(allData, existingUser)
			}

			row.Close()
		}

		return allData, errors.New("such user already exists")
	}

	allData = append(allData, userModel)

	return allData, err
}

func (UserData UserRepoRealisation) UpdateUserData(userModel models.UserModel) (models.UserModel, error) {

	id := 2
	values := make([]interface{}, 0)

	querySting := "UPDATE users SET"
	nickQuery := " WHERE nickname = $1 RETURNING u_id, nickname, fullname , email, about"
	reqQuery := ""

	values = append(values, userModel.Nickname)

	if userModel.Email != "" {
		values = append(values, userModel.Email)
		reqQuery += " " + "email = $" + strconv.Itoa(id) + ","
		id++
	}

	if userModel.Fullname != "" {
		values = append(values, userModel.Fullname)
		reqQuery += " " + "fullname = $" + strconv.Itoa(id) + ","
		id++
	}

	if userModel.About != "" {
		values = append(values, userModel.About)
		reqQuery += " " + "about = $" + strconv.Itoa(id) + ","
		id++
	}

	if len(reqQuery) > 1 {
		reqQuery = reqQuery[:len(reqQuery)-1]
	}

	var row *pgx.Row

	if len(values) == 1 {
		row = UserData.database.QueryRow("SELECT u_id, nickname, fullname , email, about FROM users WHERE nickname = $1", values[0])
	} else {
		row = UserData.database.QueryRow(querySting+reqQuery+nickQuery, values...)
	}

	userId := 0

	err := row.Scan(&userId, &userModel.Nickname, &userModel.Fullname, &userModel.Email, &userModel.About)

	return userModel, err

}

func (UserData UserRepoRealisation) GetUserData(nickname string) (models.UserModel, error) {

	userData := models.UserModel{
		Nickname: "",
		Fullname: "",
		Email:    "",
		About:    "",
	}

	row := UserData.database.QueryRow("SELECT nickname , fullname , email, about FROM users WHERE nickname = $1", nickname)

	err := row.Scan(&userData.Nickname, &userData.Fullname, &userData.Email, &userData.About)

	return userData, err
}

func (UserData UserRepoRealisation) Status() models.Status {

	statAnsw := new(models.Status)
	row := UserData.database.QueryRow("SELECT (SELECT COUNT(u_id) FROM users) as uc , (SELECT COUNT(f_id) FROM forums) AS fc , (SELECT COUNT(t_id) FROM threads) AS tc , (SELECT COUNT(m_id) FROM messages) AS mc")
	row.Scan(&statAnsw.User, &statAnsw.Forum, &statAnsw.Thread, &statAnsw.Post)

	return *statAnsw
}

func (UserData UserRepoRealisation) Clear() {
	UserData.database.Exec("DELETE FROM users;")
	UserData.database.Exec("DELETE FROM forums;")
	UserData.database.Exec("DELETE FROM threads;")
	UserData.database.Exec("DELETE FROM messages;")
	UserData.database.Exec("DELETE FROM voteThreads;")
	UserData.database.Exec("DELETE FROM forumUsers;")
}
