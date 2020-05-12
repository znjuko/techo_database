package repository

import (
	"errors"
	"github.com/jackc/pgx"
	"main/internal/models"
	"strconv"
	"time"
)

type ThreadRepoRealisation struct {
	dbLauncher *pgx.ConnPool
}

func NewThreadRepoRealisation(db *pgx.ConnPool) ThreadRepoRealisation {
	return ThreadRepoRealisation{dbLauncher: db}
}

func (Thread ThreadRepoRealisation) CreatePost(timer time.Time, forumSlug string, threadId int, posts []models.Message) ([]models.Message, error) {
	currentPosts := make([]models.Message, 0)

	tx, err := Thread.dbLauncher.Begin()

	if err != nil {
		//fmt.Println("[DEBUG] TX CREATING ERROR AT CreatePost", err)
		return nil, err
	}

	for iter, value := range posts {

		value.Thread = threadId
		value.Forum = forumSlug
		value.IsEdited = false

		var err error
		if value.Parent != 0 {

			row := tx.QueryRow("SELECT m_id , path FROM messages WHERE t_id = $2 AND m_id = $1 ", posts[iter].Parent, threadId)
			err = row.Scan(&posts[iter].Parent, &posts[iter].Path)

			if err != nil {
				tx.Rollback()
				return nil, errors.New("Parent post was created in another thread")
			}

			err = tx.QueryRow("INSERT INTO messages (date , message , parent , path , u_nickname , f_slug , t_id) VALUES ($1 , $2 , $3 , $7::BIGINT[] , $4 , $5 , $6) RETURNING date , m_id", timer, value.Message, value.Parent, value.Author, forumSlug, threadId, value.Path).Scan(&value.Created, &value.Id)
		} else {
			//"INSERT INTO messages (date , message , parent , path , u_nickname , f_slug , t_id) VALUES ($1 , $2 , $3 , ARRAY[]::BIGINT[] , $4 , $5 , $6
			err = tx.QueryRow("INSERT INTO messages (date , message , parent , path , u_nickname , f_slug , t_id) VALUES ($1 , $2 , $3 , ARRAY[]::BIGINT[] , $4 , $5 , $6) RETURNING date , m_id", timer, value.Message, value.Parent, value.Author, forumSlug, threadId).Scan(&value.Created, &value.Id)
		}

		if err != nil {

			//fmt.Println("[DEBUG] TX CREATING ERROR POST AT CreatePost", err)

			tx.Rollback()

			//fmt.Println("[DEBUG] TX ROLLBACK ERROR", err)
			return nil, errors.New("no user")
		}

		currentPosts = append(currentPosts, value)
	}
	tx.Commit()

	txFU, err := Thread.dbLauncher.Begin()

	if err != nil {
		//fmt.Println("[DEBUG] TXFU CREATING ERROR AT CreatePost", err)
		return nil, err
	}

	for _, value := range posts {
		txFU.Exec("INSERT INTO forumUsers (f_slug,u_nickname) VALUES ($1,$2) ON CONFLICT (f_slug,u_nickname) DO NOTHING ", forumSlug, value.Author)
	}

	txFU.Commit()

	Thread.dbLauncher.Exec("UPDATE forums SET message_counter = message_counter + $1 WHERE slug = $2", len(posts), forumSlug)

	return currentPosts, nil
}

func (Thread ThreadRepoRealisation) SelectThreadInfo(slug string, id int) (int, string, error) {
	threadId := 0
	forumSlug := ""
	var row *pgx.Row

	if slug != "" {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , f_slug FROM threads WHERE slug = $1", slug)
	} else {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , f_slug FROM threads WHERE t_id = $1", id)
	}

	err := row.Scan(&threadId, &forumSlug)

	if err != nil {
		//fmt.Println("[DEBUG] ERROR AT SelectThreadInfo", err)
		return 0, "", err
	}

	return threadId, forumSlug, nil
}

func (Thread ThreadRepoRealisation) GetParent(threadId int, msg []models.Message) ([]models.Message, error) {

	tx, err := Thread.dbLauncher.Begin()

	defer func () {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	} ()

	if err != nil {
		//fmt.Println("[DEBUG] TX CREATING ERROR AT CreatePost", err)
		return nil, err
	}

	for iter , _ := range msg {
		if msg[iter].Parent != 0 {
			//parentPath := make([]uint8, 0)
			row := tx.QueryRow("SELECT m_id , path FROM messages WHERE t_id = $2 AND m_id = $1 ", msg[iter].Parent, threadId)
			err = row.Scan(&msg[iter].Parent, &msg[iter].Path)

			if err != nil {
				//fmt.Println("[DEBUG] error at method CreatePost (getting parent) :", err)
				return nil, errors.New("Parent post was created in another thread")
			}

		}
	}


	return msg, nil
}

func (Thread ThreadRepoRealisation) VoteThread(nickname string, voice, threadId int, thread models.Thread) (models.Thread, error) {

	var err error
	var row *pgx.Row

	voterNick := ""

	tx, err := Thread.dbLauncher.Begin()

	defer func () {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	} ()

	if thread.Slug != "" {
		row = tx.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE slug = $1", thread.Slug)
	} else {
		row = tx.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE t_id = $1", threadId)
	}

	var forumSlug *string
	err = row.Scan(&thread.Id, &forumSlug, &thread.Author, &thread.Forum, &thread.Created, &thread.Message, &thread.Title, &thread.Votes)

	if forumSlug != nil {
		thread.Slug = *forumSlug
	}

	if err != nil {
		//fmt.Println(err)
		return thread, err
	}

	voted := 0
	row = tx.QueryRow("SELECT counter , u_nickname FROM voteThreads WHERE t_id = $1 AND u_nickname = $2", thread.Id, nickname)
	row.Scan(&voted, &voterNick)

	if voice > 0 {

		if voted != 1 {

			voteCounter := 1

			if voted == 0 {
				_, err = tx.Exec("INSERT INTO voteThreads (t_id , u_nickname, counter) VALUES ($1,$2,$3)", thread.Id, nickname, 1)
				voteCounter = 1

			} else {
				_, err = tx.Exec("UPDATE voteThreads SET counter = $3 WHERE t_id = $1 AND u_nickname = $2", thread.Id, voterNick, 1)
				voteCounter = 2
			}

			if err != nil {
				//fmt.Println("[DEBUG] error at method VoteThread (voting from err) :", err)
				return thread, err
			}

			row = tx.QueryRow("UPDATE threads SET votes = votes + $2 WHERE t_id = $1 RETURNING votes", thread.Id, voteCounter)
			err = row.Scan(&thread.Votes)

		}
	} else {
		if voted != -1 {

			voteCounter := 0

			if voted == 0 {
				_, err = tx.Exec("INSERT INTO voteThreads (t_id , u_nickname, counter) VALUES ($1,$2,$3)", thread.Id, nickname, -1)
				voteCounter = 1

			} else {
				_, err = tx.Exec("UPDATE voteThreads SET counter = $3 WHERE t_id = $1 AND u_nickname = $2", thread.Id, voterNick, -1)
				voteCounter = 2
			}

			if err != nil {
				//fmt.Println("[DEBUG] error at method VoteThread (voting from err) :", err)
				return thread, err
			}

			row = tx.QueryRow("UPDATE threads SET votes = votes - $2 WHERE t_id = $1 RETURNING votes", thread.Id, voteCounter)
			err = row.Scan(&thread.Votes)

		}
	}

	return thread, err

}

func (Thread ThreadRepoRealisation) GetThread(threadId int, thread models.Thread) (models.Thread, error) {

	var row *pgx.Row

	if thread.Slug != "" {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE slug = $1", thread.Slug)
	} else {
		row = Thread.dbLauncher.QueryRow("SELECT t_id , slug , u_nickname , f_slug , date , message , title , votes FROM threads WHERE t_id = $1", threadId)
	}

	var threadSlug *string
	err := row.Scan(&thread.Id, &threadSlug, &thread.Author, &thread.Forum, &thread.Created, &thread.Message, &thread.Title, &thread.Votes)

	if threadSlug != nil {
		thread.Slug = *threadSlug
	}

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

	var data *pgx.Rows
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
		whereQuery = " WHERE M.path[1] IN (SELECT m_id FROM messages WHERE parent = 0 AND t_id = $1 "

		if order != "DESC" {
			orderQuery = " ORDER BY M.path , M.m_id  "
		} else {
			orderQuery = " ORDER BY M.path[1] " + order + " , M.path , M.m_id  "
		}

		if limit != 0 {
			if since != 0 {
				valueCounter++
				whereQuery += "AND path[1] " + ranger + "(SELECT path[1] FROM messages WHERE t_id = $1 AND m_id = $2)" + " "
				selectValues = append(selectValues, since)
				sinceHitted = false
			}
			valueCounter++
			whereQuery += "ORDER BY m_id " + order + " LIMIT $" + strconv.Itoa(valueCounter) + ") "
			selectValues = append(selectValues, limit)
		}

		if since != 0 && sinceHitted {
			valueCounter++
			whereQuery += " AND M.path " + ranger + "(SELECT path FROM messages WHERE t_id = $1 AND m_id = $" + strconv.Itoa(valueCounter) + ") "
			selectValues = append(selectValues, since)
		}

		additionalWhere += " "
	}

	//var explain *string
	//fmt.Println(sortType, selectValues , selectQuery+whereQuery+additionalWhere+orderQuery+limitQuery)
	//errExplain ,_ := Thread.dbLauncher.Query("EXPLAIN ANALYZE "+selectQuery+whereQuery+additionalWhere+orderQuery+limitQuery, selectValues...)
	//fmt.Print("[DEBUG EXPLAIN] explain :")
	//for errExplain.Next() {
	//	errExplain.Scan(&explain)
	//	fmt.Println(*explain)
	//}
	//errExplain.Close()

	data, err = Thread.dbLauncher.Query(selectQuery+whereQuery+additionalWhere+orderQuery+limitQuery, selectValues...)

	if err != nil {
		return nil, err
	}

	if data != nil {

		for data.Next() {
			msg := new(models.Message)
			err = data.Scan(&msg.Id, &msg.Created, &msg.Message, &msg.IsEdited, &msg.Parent, &msg.Author, &msg.Thread, &msg.Forum)

			if err != nil {
				//fmt.Println(err)
			}

			messages = append(messages, *msg)
		}

		data.Close()
	}

	if len(messages) == 0 {
		trow := Thread.dbLauncher.QueryRow("SELECT t_id , slug FROM threads WHERE t_id = $1", selectValues[0])

		var threadId *int64
		var threadSlug *string

		if err = trow.Scan(&threadId, &threadSlug); err != nil {
			//fmt.Println(err)
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
	var threadRow *pgx.Rows
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
		//if err != nil {
		//	fmt.Println(err)
		//}
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

	newThreadRow := Thread.dbLauncher.QueryRow(updateRow+setRow+whereCase+returningRow, queryValues...)

	err = newThreadRow.Scan(&newThread.Id, &newThread.Slug, &newThread.Author, &newThread.Forum, &newThread.Created, &newThread.Message, &newThread.Title, &newThread.Votes)

	if err != nil {
		return newThread, err
	}

	return newThread, nil

}
