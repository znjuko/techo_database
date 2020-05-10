package threads

import (
	"main/internal/models"
	"time"
)

type ThreadsRepo interface {
	CreatePost(time.Time, string, int, []models.Message) ([]models.Message, error)
	VoteThread(string, int, int, models.Thread) (models.Thread, error)
	GetThread(int, models.Thread) (models.Thread, error)
	GetPostsSorted(string, int, int, int, string, bool) ([]models.Message, error)
	UpdateThread(string, int, models.Thread) (models.Thread, error)
	GetParent(int, models.Message) (models.Message, error)
	SelectThreadInfo(string, int) (int, string, error)
}
