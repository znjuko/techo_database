package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"main/internal/models"
	"strconv"
	"time"
)

type ThreadRepoRealisation struct {
	dbLauncher *sql.DB
}

func NewThreadRepoRealisation(db *sql.DB) ThreadRepoRealisation {
	return ThreadRepoRealisation{dbLauncher: db}
}

func (Thread ThreadRepoRealisation) CreatePost(slug string, id int, posts []models.Message) ([]models.Message, error) {
	threadId := 0
	forumSlug := ""
	var row *sql.Row

	t := time.Now()

	if slug != "" {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , f_slug FROM threads WHERE slug = $1", slug)
	} else {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , f_slug FROM threads WHERE t_id = $1", id)
	}

	err := row.Scan(&threadId, &forumSlug)

	if err != nil {
		return nil, err
	}

	currentPosts := make([]models.Message, 0)

	for _, value := range posts {

		if value.Parent == 0 {
			authorId := 0
			value.Thread = threadId
			value.Forum = forumSlug

			row = Thread.dbLauncher.QueryRow("SELECT u_id , nickname FROM users WHERE nickname = $1", value.Author)
			err = row.Scan(&authorId, &value.Author)

			if err != nil {
				return []models.Message{value}, errors.New("no user")
			}
			value.IsEdited = false
			row = Thread.dbLauncher.QueryRow("INSERT INTO messages (date , message , parent , path , u_nickname , f_slug , t_id) VALUES ($1 , $2 , $3 , ARRAY[]::BIGINT[] , $4 , $5 , $6) RETURNING date , m_id", t, value.Message, value.Parent, value.Author, forumSlug, threadId)
			err = row.Scan(&value.Created, &value.Id)
			currentPosts = append(currentPosts, value)
		} else {
			parentPath := make([]uint8, 0)
			row = Thread.dbLauncher.QueryRow("SELECT m_id , path FROM messages WHERE m_id = $1 AND t_id = $2", value.Parent, threadId)
			err = row.Scan(&value.Parent, &parentPath)

			if err != nil {
				fmt.Println("[DEBUG] error at method CreatePost (getting parent) :", err)
				return nil, errors.New("Parent post was created in another thread")
			}

			authorId := 0
			value.Thread = threadId
			value.Forum = forumSlug
			value.IsEdited = false

			row = Thread.dbLauncher.QueryRow("SELECT u_id , nickname FROM users WHERE nickname = $1", value.Author)
			err = row.Scan(&authorId, &value.Author)

			if err != nil {
				return []models.Message{value}, errors.New("no user")
			}
			dRow, err := Thread.dbLauncher.Query("INSERT INTO messages (date , message , parent , path , u_nickname , f_slug , t_id) VALUES ($1 , $2 , $3 , $7::BIGINT[] , $4 , $5 , $6) RETURNING date , m_id", t, value.Message, value.Parent, value.Author, forumSlug, threadId, parentPath)

			if err != nil {
				fmt.Println("[DEBUG] error at method CreatePost (creating post with a parent) :", err)
			}

			if dRow != nil {
				dRow.Next()
				dRow.Scan(&value.Created, &value.Id)
				dRow.Close()
			}

			currentPosts = append(currentPosts, value)
		}

		Thread.dbLauncher.Exec("INSERT INTO forumUsers (f_slug , u_nickname) VALUES($1,$2)", forumSlug, value.Author)

	}

	return currentPosts, nil
}

func (Thread ThreadRepoRealisation) VoteThread(nickname string, voice, threadId int, thread models.Thread) (models.Thread, error) {

	var err error
	var row *sql.Row

	voterNick := ""

	if thread.Slug != "" {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE slug = $1", thread.Slug)
	} else {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE t_id = $1", threadId)
	}

	err = row.Scan(&thread.Id, &thread.Slug, &thread.Author, &thread.Forum, &thread.Created, &thread.Message, &thread.Title, &thread.Votes)

	if err != nil {
		return thread, err
	}

	voted := 0
	row = Thread.dbLauncher.QueryRow("SELECT counter , u_nickname FROM voteThreads WHERE t_id = $1 AND u_nickname = $2", thread.Id, nickname)
	row.Scan(&voted, &voterNick)

	if voice > 0 {

		if voted != 1 {

			voteCounter := 1

			if voted == 0 {
				_, err = Thread.dbLauncher.Exec("INSERT INTO voteThreads (t_id , u_nickname, counter) VALUES ($1,$2,$3)", thread.Id, nickname, 1)
				voteCounter = 1

			} else {
				_, err = Thread.dbLauncher.Exec("UPDATE voteThreads SET counter = $3 WHERE t_id = $1 AND u_nickname = $2", thread.Id, voterNick, 1)
				voteCounter = 2
			}

			if err != nil {
				fmt.Println("[DEBUG] error at method VoteThread (voting from err) :", err)
				return thread, err
			}

			row = Thread.dbLauncher.QueryRow("UPDATE threads SET votes = votes + $2 WHERE t_id = $1 RETURNING votes", thread.Id, voteCounter)
			err = row.Scan(&thread.Votes)

		}
	} else {
		if voted != -1 {

			voteCounter := 0

			if voted == 0 {
				_, err = Thread.dbLauncher.Exec("INSERT INTO voteThreads (t_id , u_nickname, counter) VALUES ($1,$2, $3)", thread.Id, nickname, -1)
				voteCounter = 1

			} else {
				_, err = Thread.dbLauncher.Exec("UPDATE voteThreads SET counter = $3 WHERE t_id = $1 AND u_nickname = $2", thread.Id, voterNick, -1)
				voteCounter = 2
			}

			if err != nil {
				fmt.Println("[DEBUG] error at method VoteThread (voting from err) :", err)
				return thread, err
			}

			row = Thread.dbLauncher.QueryRow("UPDATE threads SET votes = votes - $2 WHERE t_id = $1 RETURNING votes", thread.Id, voteCounter)
			err = row.Scan(&thread.Votes)

		}
	}

	return thread, err

}

func (Thread ThreadRepoRealisation) GetThread(threadId int, thread models.Thread) (models.Thread, error) {

	var row *sql.Row

	if thread.Slug != "" {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE slug = $1", thread.Slug)
	} else {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE t_id = $1", threadId)
	}

	err := row.Scan(&thread.Id, &thread.Slug, &thread.Author, &thread.Forum, &thread.Created, &thread.Message, &thread.Title, &thread.Votes)

	if err != nil {
		return thread, err
	}

	return thread, nil
}

func (Thread ThreadRepoRealisation) GetPostsSorted(slug string, threadId int, limit int, since int, sortType string, desc bool) ([]models.Message, error) {

	ranger := ">"
	order := "ASC"
	if desc {
		order = "DESC"
		ranger = "<"
	}

	selectQuery := "SELECT m_id , date , message , edit , parent , u_nickname , t_id , f_slug FROM messages "
	whereQuery := " "
	orderQuery := " ORDER BY m_id " + order + " "
	limitQuery := " "
	additionalWhere := ""
	selectValues := make([]interface{}, 0)
	valueCounter := 1

	var err error

	if slug != "" {
		trow := Thread.dbLauncher.QueryRow("SELECT t_id FROM threads WHERE slug = $1 ", slug)

		if err = trow.Scan(&threadId); err != nil {
			return nil, err
		}
	}

	whereQuery += "WHERE t_id = $1"
	selectValues = append(selectValues, threadId)

	var data *sql.Rows
	messages := make([]models.Message, 0)

	switch sortType {
	case "flat":
		if since != 0 {
			valueCounter++
			additionalWhere += " AND m_id " + ranger + "$" + strconv.Itoa(valueCounter) + " "
			selectValues = append(selectValues, since)
		}

		if limit != 0 {
			valueCounter++
			limitQuery += " LIMIT $" + strconv.Itoa(valueCounter) + " "
			selectValues = append(selectValues, limit)
		}

	case "tree":
		orderQuery = " ORDER BY path[1] " + order + " , path " + order + " "

		if since != 0 {
			valueCounter++
			additionalWhere += " AND path " + ranger + "(SELECT path FROM messages WHERE t_id = $1 AND m_id = $" + strconv.Itoa(valueCounter) + ") "
			selectValues = append(selectValues, since)
		}

		if limit != 0 {
			valueCounter++
			limitQuery += " LIMIT $" + strconv.Itoa(valueCounter) + " "
			selectValues = append(selectValues, limit)
		}

	case "parent_tree":
		sinceHitted := true
		selectQuery = "SELECT M.m_id , M.date , M.message , M.edit , M.parent , M.u_nickname , M.t_id , M.f_slug FROM messages AS M "
		whereQuery = "WHERE M.t_id = $1"

		orderQuery = " ORDER BY M.path[1] " + order + " , M.path  "

		if limit != 0 {
			whereQuery += " AND M.path[1] IN (SELECT DISTINCT path[1] FROM messages WHERE t_id = $1 "
			if since == 0 {
				whereQuery += "AND parent = 0 "
			} else {
				valueCounter++
				whereQuery += "AND path[1] " + ranger + "(SELECT path[1] FROM messages WHERE t_id = $1 AND m_id = $2)" + " "
				selectValues = append(selectValues, since)
				sinceHitted = false
			}
			valueCounter++
			whereQuery += "ORDER BY path[1] " + order + " LIMIT $" + strconv.Itoa(valueCounter) + ") "
			selectValues = append(selectValues, limit)
		}

		if since != 0 && sinceHitted {
			valueCounter++
			whereQuery += " AND M.path " + ranger + "(SELECT path FROM messages WHERE t_id = $1 AND m_id = $" + strconv.Itoa(valueCounter) + ") "
			selectValues = append(selectValues, since)
		}

		additionalWhere += " "
	}

	data, err = Thread.dbLauncher.Query(selectQuery+whereQuery+additionalWhere+orderQuery+limitQuery, selectValues...)

	if err != nil {
		return nil, err
	}


	if data != nil {

		for data.Next() {
			msg := new(models.Message)
			err = data.Scan(&msg.Id, &msg.Created, &msg.Message, &msg.IsEdited, &msg.Parent, &msg.Author, &msg.Thread, &msg.Forum)

			if err != nil {
				fmt.Println(err)
			}

			messages = append(messages, *msg)
		}

		data.Close()
	}

	if len(messages) == 0 {
		trow := Thread.dbLauncher.QueryRow("SELECT slug FROM threads WHERE t_id = $1", selectValues[0])

		if err = trow.Scan(&slug); err != nil {
			fmt.Println(err)
			return nil, err
		}
	}

	return messages, err

}

func (Thread ThreadRepoRealisation) UpdateThread(slug string, threadId int, newThread models.Thread) (models.Thread, error) {

	whereCase := ""
	queryValues := make([]interface{}, 0)
	queryOrder := 2

	if slug != "" {
		whereCase = " WHERE slug = $1 "
		queryValues = append(queryValues, slug)
	} else {
		whereCase = " WHERE t_id = $1 "
		queryValues = append(queryValues, threadId)
	}

	var err error
	var threadRow *sql.Rows
	defer func() {
		if threadRow != nil {
			threadRow.Close()
		}
	}()

	if newThread.Title == "" && newThread.Message == "" {
		threadRow, err = Thread.dbLauncher.Query("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads "+whereCase, queryValues...)

		if err != nil || threadRow == nil {
			return newThread, err
		}

		threadRow.Next()
		err = threadRow.Scan(&newThread.Id, &newThread.Slug, &newThread.Author, &newThread.Forum, &newThread.Created, &newThread.Message, &newThread.Title, &newThread.Votes)
		if err != nil {
			fmt.Println(err)
		}
		threadRow.Close()

		return newThread, err
	}

	updateRow := "UPDATE threads SET "
	returningRow := " RETURNING t_id , slug , u_nickname , f_slug , date , message , title , votes "
	setRow := ""

	if newThread.Message != "" {
		setRow += " message = $" + strconv.Itoa(queryOrder) + ","
		queryValues = append(queryValues, newThread.Message)
		queryOrder++
	}

	if newThread.Title != "" {
		setRow += " title = $" + strconv.Itoa(queryOrder) + ","
		queryValues = append(queryValues, newThread.Title)
		queryOrder++
	}

	setRow = setRow[:len(setRow)-1]

	threadRow, err = Thread.dbLauncher.Query(updateRow+setRow+whereCase+returningRow, queryValues...)

	if err != nil {
		return newThread, err
	}

	threadRow.Next()
	err = threadRow.Scan(&newThread.Id, &newThread.Slug, &newThread.Author, &newThread.Forum, &newThread.Created, &newThread.Message, &newThread.Title, &newThread.Votes)

	if err != nil {
		return newThread, err
	}

	return newThread, nil

}
