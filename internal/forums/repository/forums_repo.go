package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"main/internal/models"
	"strconv"
)

type ForumRepoRealisation struct {
	dbLauncher *sql.DB
}

func NewForumRepoRealisation(db *sql.DB) ForumRepoRealisation {
	return ForumRepoRealisation{dbLauncher: db}
}

func (Forum ForumRepoRealisation) CreateNewForum(forum models.Forum) (models.Forum, error) {

	userId := 0

	row := Forum.dbLauncher.QueryRow("SELECT u_id , nickname FROM users WHERE nickname = $1", forum.User)

	err := row.Scan(&userId, &forum.User)

	if err != nil {
		fmt.Println("[DEBUG] error at method CreateNewForum (scan of existing user) :", err)
		return forum, err
	}

	_, err = Forum.dbLauncher.Exec("INSERT INTO forums (slug , title, u_nickname) VALUES($1 , $2 , $3)", forum.Slug, forum.Title, forum.User)

	if err != nil {
		row := Forum.dbLauncher.QueryRow("SELECT u_nickname , title , slug FROM forums WHERE slug = $1;", forum.Slug)

		row.Scan(&forum.User, &forum.Title, &forum.Slug)
		return forum, err
	}

	return forum, nil
}

func (Forum ForumRepoRealisation) GetForum(slug string) (models.Forum, error) {

	forumData := new(models.Forum)
	row := Forum.dbLauncher.QueryRow("SELECT slug , title, u_nickname FROM forums WHERE slug = $1", slug)

	err := row.Scan(&forumData.Slug, &forumData.Title, &forumData.User)

	if err != nil {
		fmt.Println("[DEBUG] error at method GetForum (scan of forum basic data) :", err)
		return *forumData, err
	}

	row = Forum.dbLauncher.QueryRow("SELECT (SELECT COUNT(DISTINCT t_id) FROM threads WHERE f_slug = $1) , (SELECT COUNT(DISTINCT m_id) FROM messages WHERE f_slug = $1)  ", forumData.Slug)

	err = row.Scan(&forumData.Threads, &forumData.Posts)

	if err != nil {
		fmt.Println("[DEBUG] error at method GetForum (scan of threads counter) :", err)
	}

	return *forumData, nil
}

func (Forum ForumRepoRealisation) CreateThread(thread models.Thread) (models.Thread, error) {

	userId := int64(0)
	var time *string
	insertValues := make([]interface{}, 0)
	valuesCounter := 4
	valuesQuery := " VALUES($1 ,$2, $3, $4,"
	insertQuery := "INSERT INTO threads "
	insertColumns := "(message , title , u_nickname , f_slug ,"
	returningQuery := " RETURNING date , t_id"

	row := Forum.dbLauncher.QueryRow("SELECT u_id , nickname FROM users WHERE nickname = $1", thread.Author)

	err := row.Scan(&userId, &thread.Author)

	if err != nil {
		fmt.Println("[DEBUG] error at method CreateThread (scan of existing user) :", err)
		return thread, err
	}

	row = Forum.dbLauncher.QueryRow("SELECT slug FROM forums WHERE slug = $1", thread.Forum)

	err = row.Scan(&thread.Forum)

	if err != nil {
		fmt.Println("[DEBUG] error at method CreateThread (scan of existing forum) :", err)
		return thread, err
	}

	insertValues = append(insertValues, thread.Message, thread.Title, thread.Author, thread.Forum)

	if thread.Slug != "" {
		insertColumns += " slug,"
		insertValues = append(insertValues, thread.Slug)
		valuesCounter++
		valuesQuery += " $" + strconv.Itoa(valuesCounter) + ","
	}

	if thread.Created != "" {
		insertColumns += " date,"
		insertValues = append(insertValues, thread.Created)
		valuesCounter++
		valuesQuery += " $" + strconv.Itoa(valuesCounter) + ","
	}

	insertColumns = insertColumns[:len(insertColumns)-1] + ")"
	valuesQuery = valuesQuery[:len(valuesQuery)-1] + ")"

	row = Forum.dbLauncher.QueryRow(insertQuery+insertColumns+valuesQuery+returningQuery, insertValues...)

	err = row.Scan(&time, &thread.Id)

	if time != nil {
		thread.Created = *time
	}

	if err != nil {
		fmt.Println("[DEBUG] error at method CreateThread (creating new forum) :", err)
		row = Forum.dbLauncher.QueryRow("SELECT u_nickname , date ,f_slug , t_id , message , slug , title , votes FROM threads WHERE slug = $1", thread.Slug)
		err = row.Scan(&thread.Author, &thread.Created, &thread.Forum, &thread.Id, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
		return thread, errors.New("thread already exist")
	}

	Forum.dbLauncher.Exec("INSERT INTO forumUsers (f_slug,u_nickname) VALUES($1,$2)", thread.Forum, thread.Author)
	return thread, nil
}

func (Forum ForumRepoRealisation) GetThreads(forum models.Forum, limit int, since string, sort bool) ([]models.Thread, error) {

	var err error
	orderStatus := "DESC"
	sorter := "<"

	if !sort {
		sorter = ">"
		orderStatus = "ASC"
	}

	var rowThreads *sql.Rows
	selectRow := "SELECT t_id , date , message , title , votes , slug , f_slug , u_nickname FROM threads T "
	if since != "" {
		sinceStatus := "WHERE date" + sorter + "=$2" + " "
		rowThreads, err = Forum.dbLauncher.Query(selectRow+sinceStatus+"AND f_slug = $3 ORDER BY date "+orderStatus+" LIMIT $1", limit, since, forum.Slug)
	} else {
		rowThreads, err = Forum.dbLauncher.Query(selectRow+"WHERE f_slug = $2 "+"ORDER BY date "+orderStatus+" LIMIT $1", limit, forum.Slug)
	}

	if err != nil {
		fmt.Println("[DEBUG] error at method GetThreads (scanning slug of a forum) :", err)
		return nil, err
	}

	threads := make([]models.Thread, 0)

	if rowThreads != nil {

		for rowThreads.Next() {
			thread := new(models.Thread)

			err = rowThreads.Scan(&thread.Id, &thread.Created, &thread.Message, &thread.Title, &thread.Votes, &thread.Slug, &thread.Forum, &thread.Author)

			if err != nil {
				fmt.Println("[DEBUG] error at method GetThreads (scanning slug of a forum) :", err)
				return nil, err
			}

			threads = append(threads, *thread)
		}

		rowThreads.Close()
	}

	if len(threads) == 0 {
		row := Forum.dbLauncher.QueryRow("SELECT slug FROM forums WHERE slug = $1", forum.Slug)
		err = row.Scan(&forum.Slug)
	}

	return threads, err
}

func (Forum ForumRepoRealisation) GetForumUsers(slug string, limit int, since string, desc bool) ([]models.UserModel, error) {

	var err error
	var row *sql.Rows

	order := "DESC"
	ranger := "<"
	users := make([]models.UserModel, 0)

	if !desc {
		order = "ASC"
		ranger = ">"
	}

	selectRow := "SELECT FU.u_nickname , U.fullname, U.email , U.about FROM forumUsers FU INNER JOIN Users U ON(U.nickname=FU.u_nickname) WHERE FU.f_slug = $1 "
	if since != "" {
		if limit == 0 {
			row, err = Forum.dbLauncher.Query(selectRow+"AND FU.u_nickname "+ranger+" $2 ORDER BY FU.u_nickname "+order, slug, since)
		} else {
			row, err = Forum.dbLauncher.Query(selectRow+" AND FU.u_nickname "+ranger+" $3 ORDER BY FU.u_nickname "+order+" LIMIT $2", slug, limit, since)
		}
	} else {
		if limit == 0 {
			row, err = Forum.dbLauncher.Query(selectRow+" ORDER BY FU.u_nickname "+order, slug)
		} else {
			row, err = Forum.dbLauncher.Query(selectRow+" ORDER BY FU.u_nickname "+order+" LIMIT $2", slug, limit)
		}
	}

	if err != nil {
		fmt.Println("[DEBUG] error at method GetForumUsers (selecting users) :", err)
		return nil, err
	}

	if row != nil {
		for row.Next() {
			user := new(models.UserModel)
			err = row.Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About)

			if err != nil {
				fmt.Println("[DEBUG] error at method GetForumUsers (selecting users) :", err)
				return nil, err
			}

			users = append(users, *user)
		}

		row.Close()
	}

	if len(users) == 0 {
		frow := Forum.dbLauncher.QueryRow("SELECT slug FROM forums WHERE slug = $1", slug)
		err = frow.Scan(&slug)

		if err != nil {
			return nil , err
		}
	}

	return users, nil

}
