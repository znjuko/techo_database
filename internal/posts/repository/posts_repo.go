package repository

import (
	"database/sql"
	"fmt"
	"main/internal/models"
)

type PostRepoRealisation struct {
	dbLauncher *sql.DB
}

func NewPostRepoRealisation(db *sql.DB) PostRepoRealisation {
	return PostRepoRealisation{dbLauncher: db}
}

func (PostRepo PostRepoRealisation) GetPost(id int, flags []string) (models.AllPostData, error) {
	msg := new(models.Message)
	answer := models.AllPostData{}
	row := PostRepo.dbLauncher.QueryRow("SELECT m_id , date , message , edit , parent , t_id , u_nickname , f_slug FROM messages WHERE m_id = $1", id)
	err := row.Scan(&msg.Id, &msg.Created, &msg.Message, &msg.IsEdited, &msg.Parent, &msg.Thread, &msg.Author, &msg.Forum)

	if err != nil {
		return answer, err
	}

	answer.Post = msg
	for _, value := range flags {
		switch value {
		case "user":
			author := new(models.UserModel)
			row = PostRepo.dbLauncher.QueryRow("SELECT nickname , fullname , email, about FROM users WHERE nickname = $1", msg.Author)
			err = row.Scan(&author.Nickname, &author.Fullname, &author.Email, &author.About)

			if err != nil {
				fmt.Println(err, "can't find a user")
			}

			answer.Author = author

		case "forum":
			forum := new(models.Forum)
			row = PostRepo.dbLauncher.QueryRow("SELECT slug , title , u_nickname FROM forums WHERE slug= $1", msg.Forum)

			err = row.Scan( &forum.Slug, &forum.Title, &forum.User )

			if err != nil {
				fmt.Println(err, "can't find a forum")
			}

			row = PostRepo.dbLauncher.QueryRow("SELECT (SELECT COUNT(DISTINCT t_id) FROM threads  WHERE f_slug = $1) AS thread_counter , (SELECT COUNT(DISTINCT m_id) FROM messages WHERE f_slug = $1) as msg_counter", forum.Slug)

			err = row.Scan(&forum.Threads, &forum.Posts)

			answer.Forum = forum

		case "thread":
			thread := new(models.Thread)
			row = PostRepo.dbLauncher.QueryRow("SELECT t_id , date , message , title , votes , slug , u_nickname , f_slug FROM threads WHERE t_id = $1", msg.Thread)
			err = row.Scan(&thread.Id, &thread.Created, &thread.Message, &thread.Title, &thread.Votes, &thread.Slug, &thread.Author, &thread.Forum)

			if err != nil {
				fmt.Println(err, "can't find a thread")
			}

			answer.Thread = thread
		}
	}

	return answer, nil
}

func (PostRepo PostRepoRealisation) UpdatePost(updateData models.Message) (models.Message, error) {

	var row *sql.Row
	if updateData.Message != "" {
		row = PostRepo.dbLauncher.QueryRow("UPDATE messages SET edit = CASE WHEN message = $1 THEN FALSE ELSE TRUE END , message = $1  WHERE m_id = $2 RETURNING m_id , date , message , edit, parent , u_nickname , f_slug , t_id", updateData.Message, updateData.Id)
	} else {
		row = PostRepo.dbLauncher.QueryRow("SELECT m_id , date , message , edit, parent , u_nickname , f_slug , t_id FROM messages WHERE m_id = $1", updateData.Id)
	}

	err := row.Scan(&updateData.Id, &updateData.Created, &updateData.Message, &updateData.IsEdited, &updateData.Parent, &updateData.Author, &updateData.Forum, &updateData.Thread)
	if err != nil {
		fmt.Println("[DEBUG] error at method UpdatePost (updating new post with message field : "+updateData.Message[:15]+") :", err)
		return updateData, err
	}

	return updateData, nil
}
