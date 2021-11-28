package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/swishcloud/goblog/storage/models"
	"github.com/swishcloud/gostudy/aesencryption"
	"github.com/swishcloud/gostudy/common"
	"github.com/swishcloud/gostudy/tx"
)

type SQLManager struct {
	Tx *tx.Tx
}

var db *sql.DB

func NewSQLManager(conn_info string) Storage {
	if db == nil {
		if d, err := tx.NewDB("postgres", conn_info); err != nil {
			panic(err)
		} else {
			db = d
		}
	}
	tx, err := tx.NewTx(db)
	if err != nil {
		panic(err)
	}
	return &SQLManager{tx}
}

func (m *SQLManager) Commit() {
	m.Tx.Commit()
}
func (m *SQLManager) Rollback() {
	m.Tx.Rollback()
}

func (m *SQLManager) GetArticle(id int, key string) *models.ArticleDto {
	r := m.Tx.QueryRow("select id,title,summary,html,content,insert_time,type,share_deadline_time,category_id,user_id,cover from article where id=$1 and is_deleted=false and is_banned=false", id)
	var (
		title             string
		summary           string
		html              string
		content           string
		insertTime        time.Time
		articleType       int
		categoryId        int
		shareDeadlineTime *time.Time
		userId            int
		cover             *string
	)
	err := r.Scan(&id, &title, &summary, &html, &content, &insertTime, &articleType, &shareDeadlineTime, &categoryId, &userId, &cover)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		panic(err)
	}
	title, err = aesencryption.Decrypt([]byte(key), title)
	if err != nil {
		panic(err)
	}
	summary, err = aesencryption.Decrypt([]byte(key), summary)
	if err != nil {
		panic(err)
	}
	html, err = aesencryption.Decrypt([]byte(key), html)
	if err != nil {
		panic(err)
	}
	content, err = aesencryption.Decrypt([]byte(key), content)
	if err != nil {
		panic(err)
	}
	return &models.ArticleDto{Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, Id: id, ArticleType: articleType, ShareDeadlineTime: shareDeadlineTime, CategoryId: categoryId, UserId: userId, Cover: cover}
}
func (m *SQLManager) GetArticles(articleType, userId int, key string, categoryId *int, secret_key string, backup_article_id *int) []models.ArticleDto {
	var typeWhere string
	var userIdWhere string
	var backupArticleIdWhere string
	var categoryWhere string
	if articleType == 0 {
		typeWhere = " and a.type!=4"
	} else {
		typeWhere = " and a.type=" + strconv.Itoa(articleType)
	}
	if userId == 0 {
		userIdWhere = " "
	} else {
		userIdWhere = " and a.user_id=" + strconv.Itoa(userId)
	}
	if backup_article_id == nil {
		backupArticleIdWhere = " and a.backup_article_id is null"
	} else {
		backupArticleIdWhere = " and a.backup_article_id=" + strconv.Itoa(*backup_article_id)
	}
	if categoryId == nil {
		categoryWhere = ""
	} else {
		categoryWhere = " and b.id=" + strconv.Itoa(*categoryId)
	}

	var rows *sql.Rows
	r, err := m.Tx.Query("select a.id,a.title,a.summary,a.html,a.content,a.insert_time,a.category_id,a.user_id,c.user_name,a.type,b.name as category_name,a.cover from article as a join category as b on a.category_id=b.id join \"user\" as c on a.user_id=c.id where title like $1 "+typeWhere+userIdWhere+backupArticleIdWhere+categoryWhere+" and a.is_deleted=false and a.is_banned=false order by a.insert_time desc", "%"+key+"%")
	if err != nil {
		panic(err)
	}
	rows = r
	defer rows.Close()

	var articles = []models.ArticleDto{}
	for rows.Next() {
		var (
			id           int
			title        string
			summary      string
			html         string
			content      string
			insertTime   time.Time
			categoryId   int
			userId       int
			userName     string
			articleType  int
			categoryName string
			cover        *string
		)
		err := rows.Scan(&id, &title, &summary, &html, &content, &insertTime, &categoryId, &userId, &userName, &articleType, &categoryName, &cover)
		if err != nil {
			panic(err)
		}
		title, err = aesencryption.Decrypt([]byte(secret_key), title)
		if err != nil {
			panic(err)
		}
		summary, err = aesencryption.Decrypt([]byte(secret_key), summary)
		if err != nil {
			panic(err)
		}
		html, err = aesencryption.Decrypt([]byte(secret_key), html)
		if err != nil {
			panic(err)
		}
		content, err = aesencryption.Decrypt([]byte(secret_key), content)
		if err != nil {
			panic(err)
		}
		if articleType == 3 {
			summary = ""
			html = ""
			content = ""
		}

		articles = append(articles, models.ArticleDto{Id: id, Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, CategoryId: categoryId, UserId: userId, UserName: userName, ArticleType: articleType, CategoryName: categoryName, Cover: cover})
	}
	return articles
}

func (m *SQLManager) NewArticle(title string, summary string, html string, content string, userId int, articleType int, shareDeadlineTime *time.Time, categoryId int, key string, cover *string, backup_article_id *int, insert_time, update_time *time.Time, remark string) int {
	summary = common.StringLimitLen(summary, 200)
	title = aesencryption.Encrypt([]byte(key), title)
	content = aesencryption.Encrypt([]byte(key), content)
	summary = aesencryption.Encrypt([]byte(key), summary)
	html = aesencryption.Encrypt([]byte(key), html)
	if articleType == 3 {
		user := m.GetUser(userId)
		if user.Level2pwd == nil {
			panic("您未设置二级密码")
		}
	}
	r := m.Tx.MustQueryRow(`INSERT INTO public.article(
		category_id, content, html, insert_time,update_time, is_banned, is_deleted, title, type,share_deadline_time,user_id, cover, summary,backup_article_id,remark)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id;`,
		categoryId, content, html, insert_time, update_time, false, false, title, articleType, shareDeadlineTime, userId, cover, summary, backup_article_id, remark)
	id := 0
	r.MustScan(&id)
	return id
}
func (m *SQLManager) UpdateArticle(id int, title string, summary string, html string, content string, articleType int, shareDeadlineTime *time.Time, categoryId, key string, userId int, cover *string) {
	summary = common.StringLimitLen(summary, 200)
	title = aesencryption.Encrypt([]byte(key), title)
	content = aesencryption.Encrypt([]byte(key), content)
	summary = aesencryption.Encrypt([]byte(key), summary)
	html = aesencryption.Encrypt([]byte(key), html)
	article := m.GetArticle(id, key)
	if articleType == 3 {
		user := m.GetUser(userId)
		if user.Level2pwd == nil {
			panic("您未设置二级密码")
		}
	} else if articleType != 1 && articleType != 2 && articleType != 5 {
		panic(fmt.Sprintf("articleType %d is invalid", articleType))
	}
	//backup
	m.NewArticle(article.Title, article.Summary, article.Html, article.Content, article.UserId, 4, article.ShareDeadlineTime, article.CategoryId, key, article.Cover, &id, &article.InsertTime, nil, "backup article")
	m.Tx.MustExec(`UPDATE public.article
	SET category_id=$1, content=$2, html=$3, title=$4, type=$5,share_deadline_time=$6, update_time=$7, cover=$8, summary=$9
	WHERE id=$10;`, categoryId, content, html, title, articleType, shareDeadlineTime, time.Now().UTC(), cover, summary, id)
}

func (m *SQLManager) GetUser(userId int) *models.UserDto {
	r := m.Tx.QueryRow(`SELECT id, user_name, level2pwd, insert_time, update_time, is_banned, op_issuer, op_userid, avatar, email
	FROM public."user" where id=$1`, userId)
	return getUser(r)
}
func getUser(r *sql.Row) *models.UserDto {
	var (
		id          int
		userName    string
		level2pwd   *string
		insert_time time.Time
		update_time *time.Time
		is_banned   bool
		op_issuer   string
		op_userid   string
		avatar      string
		email       *string
	)
	if err := r.Scan(&id, &userName, &level2pwd, &insert_time, &update_time, &is_banned, &op_issuer, &op_userid, &avatar, &email); err != nil {
		return nil
	}
	return &models.UserDto{Id: id, UserName: userName, Level2pwd: level2pwd, Insert_time: insert_time, Update_time: update_time, Is_banned: is_banned, Op_issuer: op_issuer, Op_userid: op_userid, Avatar: avatar, Email: email}
}

func (m *SQLManager) GetCategory(id int) *models.CategoryDto {
	where := " where is_deleted=false and id=$1"
	parameters := []interface{}{}
	parameters = append(parameters, id)
	categories := m.scanCategories(where, parameters...)
	if len(categories) > 0 {
		return &m.scanCategories(where, parameters...)[0]
	} else {
		return nil
	}
}
func (m *SQLManager) GetCategories(userId int, t int) []models.CategoryDto {
	where := ""
	parameters := []interface{}{}
	parameters = append(parameters, userId)
	if t == 1 {
		where = "where id in( select a.id from category as a join article as b on a.id=b.category_id  where b.type=1 and a.is_deleted=false and a.user_id=$1 group by a.id ) order by id"

	} else {
		where = "where is_deleted=false and user_id=$1 order by id"

	}
	return m.scanCategories(where, parameters...)
}
func (m *SQLManager) scanCategories(where string, args ...interface{}) []models.CategoryDto {
	rows := m.Tx.MustQuery("select id,name from category "+where, args...)
	var categoryList []models.CategoryDto
	for rows.Next() {
		var (
			id   int
			name string
		)
		if err := rows.Scan(&id, &name); err != nil {
			panic(err)
		}
		categoryList = append(categoryList, models.CategoryDto{Id: id, Name: name})
	}
	return categoryList
}
func (m *SQLManager) ArticleDelete(id, loginUserId int, key string) {
	var article = m.GetArticle(id, key)
	if article.UserId != loginUserId {
		panic("no permission")
	}
	m.Tx.MustExec("update article set is_deleted=true where id=$1", id)
}

func (m *SQLManager) CategoryDelete(categoryId int) {
	var id int
	err := m.Tx.MustQueryRow("select id from article where category_id=$1", categoryId).Scan(&id)
	if err == nil {
		panic("该分类下面有文章不能删除")
	}
	m.Tx.MustExec(`delete from category where id=$1`, categoryId)
}
func (m *SQLManager) UpdateCategory(name string, id, loginUserId int) {
	m.Tx.MustExec(`update category set name=$1 where id=$2 and user_id=$3`, name, id, loginUserId)
}

func (m *SQLManager) SetLevelTwoPwd(userId int, pwd string) {
	m.Tx.MustExec(`update public."user" set level2pwd=$1 where id=$2`, common.Md5Hash(pwd), userId)
}
func (m *SQLManager) GetUserByOP(userid, issuer string) (*models.UserDto, error) {
	r := m.Tx.QueryRow(`SELECT id, user_name, level2pwd, insert_time, update_time, is_banned, op_issuer, op_userid, avatar, email
	FROM public."user" where op_userid=$1 and op_issuer=$2`, userid, issuer)
	u := getUser(r)
	if u == nil {
		return nil, errors.New("accout not exists")
	}
	return u, nil
}
func (m *SQLManager) NewUser(username, op_issuer, op_userid, email, avatar string) {
	r := m.Tx.MustQueryRow(`INSERT INTO public."user"(
		user_name, insert_time, is_banned, op_issuer, op_userid,email, avatar)
		VALUES ($1, $2, $3, $4,$5, $6,$7) RETURNING id`, username, time.Now().UTC(), false, op_issuer, op_userid, email, avatar)
	lastInsertId := 0
	err := r.Scan(&lastInsertId)
	if err != nil {
		panic(err)
	}
	intLastId := int(lastInsertId)
	m.NewCategory("默认分类", intLastId)
}
func (m *SQLManager) NewCategory(name string, userId int) {
	m.Tx.MustExec(`INSERT INTO public.category(
		name, insert_time,  is_deleted, user_id)
		VALUES ($1,$2,$3,$4);`, name, time.Now().UTC(), false, userId)
}

func (m *SQLManager) NewFriendlyLink(website_name, website_url, description, friendly_link_page_url string) {
	id := uuid.New()
	m.Tx.MustExec(`INSERT INTO public.friendly_link(
		id, website_name,description, website_url, friendly_link_page_url, insert_time, access_time, is_approved, is_deleted)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9);`, id, website_name, description, website_url, friendly_link_page_url, time.Now().UTC(), nil, false, false)
}

func (m *SQLManager) GetFriendlyLinks() ([]models.FriendlyLink, error) {
	rows, err := m.Tx.Query(`SELECT id, description, website_url, friendly_link_page_url, insert_time, access_time, is_approved, is_deleted, website_name
	FROM public.friendly_link where is_deleted=false order by access_time desc;`)
	if err != nil {
		return nil, err
	}
	result := []models.FriendlyLink{}
	for rows.Next() {
		item := models.FriendlyLink{}
		err = rows.Scan(&item.Id, &item.Description, &item.Website_url, &item.Friendly_link_page_url, &item.Insert_time, &item.Access_time, &item.Is_approved, &item.Is_deleted, &item.Website_name)
		result = append(result, item)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (m *SQLManager) FreshFriendlyLinkAccessTime(id string) {
	m.Tx.MustExec(`update public.friendly_link set access_time=$1 where id=$2`, time.Now().UTC(), id)
}
func (m *SQLManager) SetFriendlyLinkActiveStatus(id string, active bool) {
	m.Tx.MustExec(`update public.friendly_link set is_approved=$1 where id=$2`, active, id)
}

func (m *SQLManager) DeleteFriendlyLink(id string) {
	m.Tx.MustExec(`update public.friendly_link set is_deleted=true where id=$1`, id)
}
