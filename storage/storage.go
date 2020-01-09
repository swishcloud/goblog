package storage

import "github.com/swishcloud/goblog/storage/models"

type Storage interface {
	GetArticle(id int, key string) *models.ArticleDto
	GetArticles(articleType, userId int, key string, categoryName string, secret_key string) []models.ArticleDto
	NewArticle(title string, summary string, html string, content string, userId int, articleType int, categoryId int, key string, cover *string, backup_article_id *int) int
	UpdateArticle(id int, title string, summary string, html string, content string, articleType int, categoryId, key string, userId int, cover *string)
	GetUser(userId int) *models.UserDto
	GetCategory(id int) *models.CategoryDto
	GetCategories(userId int, t int) []models.CategoryDto
	ArticleDelete(id, loginUserId int, key string)
	CategoryDelete(categoryId int)
	UpdateCategory(name string, id, loginUserId int)
	SetLevelTwoPwd(userId int, pwd string)
	GetUserByOP(userid, issuer string) (*models.UserDto, error)
	NewUser(username, op_issuer, op_userid, email, avatar string)
	NewCategory(name string, userId int)
	Commit()
	Rollback()
}
