package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/mysql"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/swishcloud/gostudy/keygenerator"
	"github.com/swishcloud/gostudy/tx"

	"github.com/swishcloud/goblog/common"
	"github.com/swishcloud/goblog/storage/models"
	"github.com/swishcloud/gostudy/aesencryption"
	externalCommon "github.com/swishcloud/gostudy/common"
)

type SQLManager struct {
	Tx *tx.Tx
}

var db *sql.DB

func NewSQLManager(conn_info string) Storage {
	if db == nil {
		d, err := sql.Open("mysql", conn_info)
		if err != nil {
			panic(err)
		}
		db = d
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

func (m *SQLManager) GetArticle(id int) *models.ArticleDto {
	r := m.Tx.QueryRow("select id,title,summary,html,content,insertTime,type,categoryId,userId,cover from article where id=?", id)
	var (
		title       string
		summary     string
		html        string
		content     string
		insertTime  string
		articleType int
		categoryId  int
		userId      int
		cover       *string
	)
	if err := r.Scan(&id, &title, &summary, &html, &content, &insertTime, &articleType, &categoryId, &userId, &cover); err != nil {
		return nil
	}
	insertTime = common.ConvUtcToLocal(insertTime, "2006-01-02 15:04:05", "2006-01-02 15:04")
	return &models.ArticleDto{Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, Id: id, ArticleType: articleType, CategoryId: categoryId, UserId: userId, Cover: cover}
}
func (m *SQLManager) GetArticles(articleType, userId int, key string, withLockedContext bool, categoryName string) []models.ArticleDto {
	var typeWhere string
	var userIdWhere string
	if articleType == 0 {
		typeWhere = ""
	} else {
		typeWhere = " and a.type=" + strconv.Itoa(articleType)
	}

	if userId == 0 {
		userIdWhere = ""
	} else {
		userIdWhere = " and a.userId=" + strconv.Itoa(userId)
	}
	var rows *sql.Rows
	if categoryName == "" {
		r, err := m.Tx.Query("select a.id,a.title,a.summary,a.html,a.content,a.insertTime,a.categoryId,a.userId,a.type,b.name as categoryName,a.cover from article as a join category as b on a.categoryId=b.id where title like ? "+typeWhere+userIdWhere+" and type!=4 and a.isDeleted=0 and isBanned=0 order by a.insertTime desc", "%"+key+"%")
		if err != nil {
			panic(err.Error())
		}
		rows = r
	} else {
		r, err := m.Tx.Query("select a.id,a.title,a.summary,a.html,a.content,a.insertTime,a.categoryId,a.userId,a.type,b.name as categoryName,a.cover from article as a join category as b on a.categoryId=b.id where b.name=? and title like ? "+typeWhere+userIdWhere+" and type!=4 and a.isDeleted=0 and isBanned=0 order by  a.insertTime desc", categoryName, "%"+key+"%")
		if err != nil {
			panic(err.Error())
		}
		rows = r
	}
	defer rows.Close()

	var articles []models.ArticleDto
	for rows.Next() {
		var (
			id           int
			title        string
			summary      string
			html         string
			content      string
			insertTime   string
			categoryId   int
			userId       int
			articleType  int
			categoryName string
			cover        *string
		)
		if err := rows.Scan(&id, &title, &summary, &html, &content, &insertTime, &categoryId, &userId, &articleType, &categoryName, &cover); err != nil {
			panic(err)
		}
		insertTime = common.ConvUtcToLocal(insertTime, "2006-01-02 15:04:05", "2006-01-02 15:04")
		if articleType == 3 && !withLockedContext {
			content = ""
			summary = ""
		}
		articles = append(articles, models.ArticleDto{Id: id, Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, CategoryId: categoryId, UserId: userId, ArticleType: articleType, CategoryName: categoryName, Cover: cover})
	}
	return articles
}

func (m *SQLManager) NewArticle(title string, summary string, html string, content string, userId int, articleType int, categoryId int, key string, cover *string) int {
	if articleType == 3 {
		user := m.GetUser(userId)
		if user.Level2pwd == nil {
			panic("您未设置二级密码")
		}
		content = aesencryption.Encrypt([]byte(key), content)
		summary = aesencryption.Encrypt([]byte(key), summary)
		html = aesencryption.Encrypt([]byte(key), html)
	}
	summary = externalCommon.StringLimitLen(summary, 200)
	id, err := m.Tx.MustExec(`insert into article (title,summary,html,content,userId,insertTime,updateTime,isDeleted,isBanned,type,categoryId,cover)values(?,?,?,?,?,?,?,?,?,?,?,?)`, title, summary, html, content, userId, time.Now(), time.Now(), 0, 0, articleType, categoryId, cover).LastInsertId()
	if err != nil {
		panic(err)
	}
	return int(id)
}
func (m *SQLManager) UpdateArticle(id int, title string, summary string, html string, content string, articleType int, categoryId, key string, userId int, cover *string) {
	m.articleBackup(id, content, key)
	if articleType == 3 {
		user := m.GetUser(userId)
		if user.Level2pwd == nil {
			panic("您未设置二级密码")
		}
		content = aesencryption.Encrypt([]byte(key), content)
		summary = aesencryption.Encrypt([]byte(key), summary)
		html = aesencryption.Encrypt([]byte(key), html)
	} else if articleType != 1 && articleType != 2 {
		panic(fmt.Sprintf("articleType %d is invalid", articleType))
	}
	summary = externalCommon.StringLimitLen(summary, 200)
	m.Tx.MustExec(`update article set title=?,summary=?,html=?,content=?,type=?,categoryId=?,updateTime=?,cover=? where id=?`, title, summary, html, content, articleType, categoryId, time.Now(), cover, id)
}
func (m *SQLManager) articleBackup(articleId int, content string, key string) {
	m.Tx.MustExec("insert into articleBackup (content,articleId,insertTime)values(?,?,?)", aesencryption.Encrypt([]byte(key), content), articleId, time.Now())
}

func (m *SQLManager) GetUser(userId int) *models.UserDto {
	r := m.Tx.QueryRow("select id,userName,level2pwd,emailConfirmed,securityStamp,email from user where id=? and isdeleted=0 and isBanned=0", userId)
	return getUser(r)
}
func getUser(r *sql.Row) *models.UserDto {
	var (
		id             int
		userName       string
		level2pwd      *string
		emailConfirmed int
		securityStamp  *string
		email          *string
	)
	if err := r.Scan(&id, &userName, &level2pwd, &emailConfirmed, &securityStamp, &email); err != nil {
		return nil
	}
	return &models.UserDto{Id: id, UserName: userName, Level2pwd: level2pwd, EmailConfirmed: emailConfirmed, SecurityStamp: securityStamp, Email: email}
}
func (m *SQLManager) WsmessageInsert(msg string) error {
	_, err := m.Tx.Exec("insert into wsmessage (insertTime,msg,isDeleted) values(?,?,?)", time.Now(), msg, 0)
	if err != nil {
		return err
	}
	return nil
}

func (m *SQLManager) WsmessageTop() ([]models.WsmessageDto, error) {
	rows, err := m.Tx.Query("select  insertTime,msg from goblog.wsmessage where isDeleted=0 and insertTime> (UTC_TIMESTAMP() - INTERVAL 60 MINUTE) order by  insertTime desc limit 100")
	if err != nil {
		return nil, err
	}
	dtos := []models.WsmessageDto{}
	for rows.Next() {
		var (
			insertTime string
			msg        string
		)
		err := rows.Scan(&insertTime, &msg)
		if err != nil {
			return nil, err
		}
		dtos = append(dtos, models.WsmessageDto{InsertTime: common.ConvUtcToLocal(insertTime, common.TimeLayoutMysqlDateTime, common.TimeLayout2), Msg: msg})
	}
	return dtos, nil
}

func (m *SQLManager) GetCategories(userId int, t int) []models.CategoryDto {
	var rows *sql.Rows
	if t == 1 {
		r, err := m.Tx.Query("select a.id,a.name  from category where id in( select a.id from category as a join article as b on a.id=b.categoryId  where b.type=1 and a.isdeleted=0 and a.userId=? order by name group by a.id ) ", userId)
		if err != nil {
			panic(err)
		}
		rows = r
	} else {
		r, err := m.Tx.Query("select id,name from category where isdeleted=0 and userId=? order by name", userId)
		if err != nil {
			panic(err)
		}
		rows = r

	}
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
	var article = m.GetArticle(id)
	if article.UserId != loginUserId {
		panic("no permission")
	}
	m.Tx.MustExec("update article set isDeleted=1,content=?,html=?,summary=? where id=?", aesencryption.Encrypt([]byte(key), article.Content), aesencryption.Encrypt([]byte(key), article.Html), aesencryption.Encrypt([]byte(key), article.Summary), id)
}

func (m *SQLManager) CategoryDelete(categoryId int) {
	var id int
	err := m.Tx.MustQueryRow("select id from article where categoryId=?", categoryId).Scan(&id)
	if err != nil {
		panic("该分类下面有文章不能删除")
	}
	m.Tx.Exec(`delete from category where id=?`, id)
}
func (m *SQLManager) UpdateCategory(name string, id, loginUserId int) {
	m.Tx.Exec(`update category set name=? where id=? and userId=?`, name, id, loginUserId)
}

func (m *SQLManager) SetLevelTwoPwd(userId int, pwd string) {
	m.Tx.Exec("update user set level2pwd=? where id=?", common.Md5Hash(pwd), userId)
}

func InitializeDb(connInfo string) *sql.DB {
	db, _ := sql.Open("mysql", connInfo)
	err := db.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		log.Fatal(err.Error())
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"mysql",
		driver,
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Println("database is up to date")
		} else {
			log.Fatal(err.Error())
		}
	} else {
		log.Println("successfully updated database")
	}
	return db
}
func (m *SQLManager) GetUserByOP(userid, issuer string) (*models.User, error) {
	var (
		id                int
		userName          string
		accessFailedCount int
		lockoutEnd        *string
		emailConfirmed    int
		avatar            *string
	)
	r := m.Tx.QueryRow("select id,userName,accessFailedCount,lockoutEnd,emailConfirmed,avatar from user where op_userid=? and op_issuer=?", userid, issuer)
	err := r.Scan(&id, &userName, &accessFailedCount, &lockoutEnd, &emailConfirmed, &avatar)
	if err != nil {
		return nil, errors.New("账号不存在")
	}
	u := &models.User{}
	u.Avatar = avatar
	u.Id = id
	u.UserName = userName
	return u, nil
}
func (m *SQLManager) NewUser(username, op_issuer, op_userid, email string) {
	securityStamp, err := keygenerator.NewKey(32, false, false, false, false)
	if err != nil {
		panic(err)
	}
	r := m.Tx.MustExec(`insert into user (userName,insertTime,isDeleted,isBanned,accessFailedCount,securityStamp,emailConfirmed,email,op_issuer,op_userid)values(?,?,?,?,?,?,?,?,?,?)`, username, time.Now(), 0, 0, 0, securityStamp, 1, email, op_issuer, op_userid)
	lastId, err := r.LastInsertId()
	if err != nil {
		panic(err)
	}
	intLastId := int(lastId)
	m.NewCategory("默认分类", intLastId)
}
func (m *SQLManager) NewCategory(name string, userId int) {
	m.Tx.MustExec(`insert into category (name,insertTime,isDeleted,userId)values(?,?,?,?)`, name, time.Now(), 0, userId)
}
