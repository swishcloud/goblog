package storage

import (
	"time"

	"github.com/swishcloud/goblog/storage/models"
)

type Storage interface {
	GetArticle(id int, key string) *models.ArticleDto
	GetArticles(articleType, userId int, key string, categoryId *int, secret_key string, backup_article_id *int) []models.ArticleDto
	NewArticle(title string, summary string, html string, content string, userId int, articleType int, shareDeadlineTime *time.Time, categoryId int, key string, cover *string, backup_article_id *int, insert_time, update_time *time.Time, remark string) int
	UpdateArticle(id int, title string, summary string, html string, content string, articleType int, shareDeadlineTime *time.Time, categoryId, key string, userId int, cover *string)
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
	NewFriendlyLink(website_name, website_url, description, friendly_link_page_url string)
	GetFriendlyLinks() ([]models.FriendlyLink, error)
	FreshFriendlyLinkAccessTime(id string)
	SetFriendlyLinkActiveStatus(id string, active bool)
	DeleteFriendlyLink(id string)
	AddImage(related_id *string, image_type int, image_src string, cloud_url *string, user_id *int)
	UpdateImageRelatedId(image_src string, related_id string, user_id int)
	GetImage(image_src string) map[string]interface{}
	GetLocalOnlyImages() []map[string]interface{}
	UpdateImageCloudUrl(id string, cloud_url string) error
	Commit()
	Rollback()
}
