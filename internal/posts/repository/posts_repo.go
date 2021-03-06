package repository

import (
	"github.com/jackc/pgx"
	"main/internal/models"
)

type PostRepoRealisation struct {
	dbLauncher *pgx.ConnPool
}

func NewPostRepoRealisation(db *pgx.ConnPool) PostRepoRealisation {
	return PostRepoRealisation{dbLauncher: db}
}

func (PostRepo PostRepoRealisation) GetPost(id int, flags []string) (models.AllPostData, error) {
	msg := new(models.Message)
	answer := models.AllPostData{}

	tx, err := PostRepo.dbLauncher.Begin()

	if err != nil {
		tx.Rollback()
		return models.AllPostData{}, err
	}

	var row *pgx.Row
	tx.Prepare("get-msg", "SELECT m_id , date , message , edit , parent ,  u_nickname , t_id , f_slug FROM messages WHERE m_id = $1")
	if len(flags) == 0 {
		row = tx.QueryRow("get-msg", id)
	} else {
		row = tx.QueryRow("get-msg", id)
	}
	err = row.Scan(&msg.Id, &msg.Created, &msg.Message, &msg.IsEdited, &msg.Parent, &msg.Author, &msg.Thread, &msg.Forum)

	if err != nil {
		tx.Rollback()
		return answer, err
	}

	answer.Post = msg
	for _, value := range flags {
		switch value {
		case "user":
			author := new(models.UserModel)
			row = tx.QueryRow("SELECT nickname , fullname , email, about FROM users WHERE nickname = $1", msg.Author)
			err = row.Scan(&author.Nickname, &author.Fullname, &author.Email, &author.About)

			//if err != nil {
			//	fmt.Println(err, "can't find a user")
			//}

			answer.Author = author

		case "forum":
			forum := new(models.Forum)
			row = tx.QueryRow("SELECT slug , title , u_nickname, message_counter , thread_counter FROM forums WHERE slug= $1", msg.Forum)

			err = row.Scan(&forum.Slug, &forum.Title, &forum.User, &forum.Posts, &forum.Threads)

			answer.Forum = forum

		case "thread":
			thread := new(models.Thread)
			row = tx.QueryRow("SELECT t_id , date , message , title , votes , slug , u_nickname , f_slug FROM threads WHERE t_id = $1", msg.Thread)
			var threadSlug *string
			err = row.Scan(&thread.Id, &thread.Created, &thread.Message, &thread.Title, &thread.Votes, &threadSlug, &thread.Author, &thread.Forum)

			if threadSlug != nil {
				thread.Slug = *threadSlug
			}

			answer.Thread = thread
		}
	}
	tx.Commit()
	return answer, nil
}

func (PostRepo PostRepoRealisation) UpdatePost(updateData models.Message) (models.Message, error) {

	var row *pgx.Row
	if updateData.Message != "" {
		row = PostRepo.dbLauncher.QueryRow("UPDATE messages SET edit = CASE WHEN message = $1 THEN FALSE ELSE TRUE END , message = $1  WHERE m_id = $2 RETURNING m_id , date , message , edit, parent , u_nickname , t_id, f_slug", updateData.Message, updateData.Id)
	} else {
		row = PostRepo.dbLauncher.QueryRow("SELECT m_id , date , message , edit, parent , u_nickname ,  t_id ,f_slug FROM messages WHERE m_id = $1", updateData.Id)
	}

	err := row.Scan(&updateData.Id, &updateData.Created, &updateData.Message, &updateData.IsEdited, &updateData.Parent, &updateData.Author, &updateData.Thread, &updateData.Forum)
	if err != nil {
		//fmt.Println("[DEBUG] error at method UpdatePost (updating new post with message field : "+updateData.Message[:15]+") :", err)
		return updateData, err
	}

	return updateData, nil
}
