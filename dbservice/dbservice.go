package dbservice

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/github-123456/goblog/common"
	"github.com/github-123456/gostudy/aesencryption"
	externalCommon "github.com/github-123456/gostudy/common"
	"github.com/github-123456/gostudy/keygenerator"
	"github.com/github-123456/gostudy/superdb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/mysql"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/swishcloud/goblog/storage/models"
)

var db *sql.DB

func SetDb(d *sql.DB) {
	db = d
}

type dbServiceError struct {
	error string
}

func (err dbServiceError) Error() string {
	return err.error
}

func InitializeDb(connInfo string) *sql.DB {
	db, _ = sql.Open("mysql", connInfo)
	err := db.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		log.Fatal(err.Error())
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migration",
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
	SetDb(db)
	return db
}
func GetArticles(articleType, userId int, key string, withLockedContext bool, categoryName string) []ArticleDto {
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
		r, err := db.Query("select a.id,a.title,a.summary,a.html,a.content,a.insertTime,a.categoryId,a.userId,a.type,b.name as categoryName,a.cover from article as a join category as b on a.categoryId=b.id where title like ? "+typeWhere+userIdWhere+" and type!=4 and a.isDeleted=0 and isBanned=0 order by a.insertTime desc", "%"+key+"%")
		if err != nil {
			panic(err.Error())
		}
		rows = r
	} else {
		r, err := db.Query("select a.id,a.title,a.summary,a.html,a.content,a.insertTime,a.categoryId,a.userId,a.type,b.name as categoryName,a.cover from article as a join category as b on a.categoryId=b.id where b.name=? and title like ? "+typeWhere+userIdWhere+" and type!=4 and a.isDeleted=0 and isBanned=0 order by  a.insertTime desc", categoryName, "%"+key+"%")
		if err != nil {
			panic(err.Error())
		}
		rows = r
	}
	defer rows.Close()

	var articles []ArticleDto
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
		articles = append(articles, ArticleDto{Id: id, Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, CategoryId: categoryId, UserId: userId, ArticleType: articleType, CategoryName: categoryName, Cover: cover})
	}
	return articles
}

func GetCategories(userId int, t int) []CategoryDto {
	var rows *sql.Rows
	if t == 1 {
		r, err := db.Query("select a.id,a.name  from category where id in( select a.id from category as a join article as b on a.id=b.categoryId  where b.type=1 and a.isdeleted=0 and a.userId=? order by name group by a.id ) ", userId)
		if err != nil {
			panic(err)
		}
		rows = r
	} else {
		r, err := db.Query("select id,name from category where isdeleted=0 and userId=? order by name", userId)
		if err != nil {
			panic(err)
		}
		rows = r

	}
	var categoryList []CategoryDto
	for rows.Next() {
		var (
			id   int
			name string
		)
		if err := rows.Scan(&id, &name); err != nil {
			panic(err)
		}
		categoryList = append(categoryList, CategoryDto{Id: id, Name: name})
	}
	return categoryList
}
func CategoryDelete(categoryId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
		var id int
		err := tx.MustQueryRow("select id from article where categoryId=?", categoryId).Scan(&id)
		if err != nil {
			panic("该分类下面有文章不能删除")
		}
		tx.Exec(`delete from category where id=?`, id)
	}
}
func UpdateCategory(name string, id, loginUserId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
		tx.Exec(`update category set name=? where id=? and userId=?`, name, id, loginUserId)
	}
}

func SetLevelTwoPwd(userId int, pwd string) superdb.DbTask {
	return func(tx *superdb.Tx) {
		tx.Exec("update user set level2pwd=? where id=?", common.Md5Hash(pwd), userId)
	}
}
func NewArticle(title string, summary string, html string, content string, userId int, articleType int, categoryId int, key string, cover *string) superdb.DbTask {
	return func(tx *superdb.Tx) {
		if articleType == 3 {
			user := GetUser(userId)
			if user.Level2pwd == nil {
				panic("您未设置二级密码")
			}
			content = aesencryption.Encrypt([]byte(key), content)
			summary = aesencryption.Encrypt([]byte(key), summary)
			html = aesencryption.Encrypt([]byte(key), html)
		}
		summary = externalCommon.StringLimitLen(summary, 200)
		id, err := tx.MustExec(`insert into article (title,summary,html,content,userId,insertTime,updateTime,isDeleted,isBanned,type,categoryId,cover)values(?,?,?,?,?,?,?,?,?,?,?,?)`, title, summary, html, content, userId, time.Now(), time.Now(), 0, 0, articleType, categoryId, cover).LastInsertId()
		if err != nil {
			panic(err)
		}
		tx.SetValue("NewArticleLastInsertId", id)
	}
}
func UpdateArticle(id int, title string, summary string, html string, content string, articleType int, categoryId, key string, userId int, cover *string) superdb.DbTask {
	return func(tx *superdb.Tx) {
		articleBackup(id, content, key)(tx)
		if articleType == 3 {
			user := GetUser(userId)
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
		tx.MustExec(`update article set title=?,summary=?,html=?,content=?,type=?,categoryId=?,updateTime=?,cover=? where id=?`, title, summary, html, content, articleType, categoryId, time.Now(), cover, id)
	}
}

func articleBackup(articleId int, content string, key string) superdb.DbTask {
	return func(tx *superdb.Tx) {
		tx.MustExec("insert into articleBackup (content,articleId,insertTime)values(?,?,?)", aesencryption.Encrypt([]byte(key), content), articleId, time.Now())
	}
}

func GetArticle(id int) *ArticleDto {
	r := db.QueryRow("select id,title,summary,html,content,insertTime,type,categoryId,userId,cover from article where id=?", id)
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
	return &ArticleDto{Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, Id: id, ArticleType: articleType, CategoryId: categoryId, UserId: userId, Cover: cover}
}

func ArticleDelete(id, loginUserId int, key string) superdb.DbTask {
	return func(tx *superdb.Tx) {
		var article = GetArticle(id)
		if article.UserId != loginUserId {
			panic("no permission")
		}
		tx.MustExec("update article set isDeleted=1,content=?,html=?,summary=? where id=?", aesencryption.Encrypt([]byte(key), article.Content), aesencryption.Encrypt([]byte(key), article.Html), aesencryption.Encrypt([]byte(key), article.Summary), id)
	}
}

func GetUserByEmail(email string) *UserDto {
	r := db.QueryRow("select id,userName,level2pwd,emailConfirmed,securityStamp,email from user where email=? and isdeleted=0 and isBanned=0", email)
	return getUser(r)
}
func GetUser(userId int) *UserDto {
	r := db.QueryRow("select id,userName,level2pwd,emailConfirmed,securityStamp,email from user where id=? and isdeleted=0 and isBanned=0", userId)
	return getUser(r)
}
func getUser(r *sql.Row) *UserDto {
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
	return &UserDto{Id: id, UserName: userName, Level2pwd: level2pwd, EmailConfirmed: emailConfirmed, SecurityStamp: securityStamp, Email: email}
}
func ValidateEmail(email, securityStamp string) {
	r, err := db.Exec("update user set emailConfirmed=1 where email=? and emailConfirmed=0 and securityStamp=?", email, securityStamp)
	if err != nil {
		panic(err)
	}
	if n, err := r.RowsAffected(); err != nil {
		panic(err)
	} else if n == 0 {
		panic("验证失败")
	}
}
func NewCategory(name string, userId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
		tx.MustExec(`insert into category (name,insertTime,isDeleted,userId)values(?,?,?,?)`, name, time.Now(), 0, userId)
	}
}

func NewUser(username, op_issuer, op_userid, email string) superdb.DbTask {
	securityStamp := keygenerator.NewKey(32)
	return func(tx *superdb.Tx) {
		r := tx.MustExec(`insert into user (userName,insertTime,isDeleted,isBanned,accessFailedCount,securityStamp,emailConfirmed,email,op_issuer,op_userid)values(?,?,?,?,?,?,?,?,?,?)`, username, time.Now(), 0, 0, 0, securityStamp, 1, email, op_issuer, op_userid)
		lastId, err := r.LastInsertId()
		if err != nil {
			panic(err)
		}
		intLastId := int(lastId)
		NewCategory("默认分类", intLastId)(tx)
		tx.SetValue("newUserId", intLastId)
	}
}
func GetUserByOP(userid, issuer string) (*models.User, error) {
	var (
		id                int
		userName          string
		accessFailedCount int
		lockoutEnd        *string
		emailConfirmed    int
		avatar            *string
	)
	r := db.QueryRow("select id,userName,accessFailedCount,lockoutEnd,emailConfirmed,avatar from user where op_userid=? and op_issuer=?", userid, issuer)
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

func CheckUser(account, pwd string, maxAllowAccessFaildCount int) (*UserDto, error) {
	var (
		id                int
		password          string
		userName          string
		accessFailedCount int
		lockoutEnd        *string
		emailConfirmed    int
	)
	r := db.QueryRow("select id,userName, password,accessFailedCount,lockoutEnd,emailConfirmed from user where userName=? or email=?", account, account)
	err := r.Scan(&id, &userName, &password, &accessFailedCount, &lockoutEnd, &emailConfirmed)
	if err != nil {
		return nil, common.Error{fmt.Sprintf("账号不存在")}
	}

	user := GetUser(id)

	if emailConfirmed == 0 {
		return user, common.Error{"注册邮箱未激活"}
	}

	if lockoutEnd != nil {
		lockoutEndTime, err := time.Parse("2006-01-02 15:04:05", *lockoutEnd)
		if err != nil {
			return user, err
		}
		fmt.Println(lockoutEndTime.UTC(), time.Now().UTC())
		if lockoutEndTime.UTC().Before(time.Now().UTC()) {
			UnlockUser(id)
		} else {
			return user, common.Error{"您的账号已被锁定"}
		}
	}
	if !common.Md5Check(password, pwd) {
		failedCount := accessFailedCount + 1
		db.Exec("update user set accessFailedCount=? where userName=?", failedCount, account)
		if remainC := maxAllowAccessFaildCount - failedCount; remainC > 0 {
			return user, common.Error{fmt.Sprintf("密码错误，您还有%d次重试机会", remainC)}
		} else {
			LockUser(id)
			ResetAccessFailedCount(id)
			return user, common.Error{"您的账号已被锁定"}
		}
	}
	ResetAccessFailedCount(id)
	return user, nil
}

func LockUser(id int) {
	_, err := db.Exec("update user set lockoutEnd=? where id=?", time.Now().Add(time.Minute*20), id)
	if err != nil {
		panic(err)
	}
}
func UnlockUser(id int) {
	_, err := db.Exec("update user set lockoutEnd=null where id=?", id)
	if err != nil {
		panic(err)
	}
}

func ResetAccessFailedCount(id int) {
	_, err := db.Exec("update user set accessFailedCount=0  where id=?", id)
	if err != nil {
		panic(err)
	}
}

func WsmessageInsert(msg string) error {
	_, err := db.Exec("insert into wsmessage (insertTime,msg,isDeleted) values(?,?,?)", time.Now(), msg, 0)
	if err != nil {
		return err
	}
	return nil
}

func WsmessageTop() ([]WsmessageDto, error) {
	rows, err := db.Query("select  insertTime,msg from goblog.wsmessage where isDeleted=0 and insertTime> (UTC_TIMESTAMP() - INTERVAL 60 MINUTE) order by  insertTime desc limit 100")
	if err != nil {
		return nil, err
	}
	dtos := []WsmessageDto{}
	for rows.Next() {
		var (
			insertTime string
			msg        string
		)
		err := rows.Scan(&insertTime, &msg)
		if err != nil {
			return nil, err
		}
		dtos = append(dtos, WsmessageDto{InsertTime: common.ConvUtcToLocal(insertTime, common.TimeLayoutMysqlDateTime, common.TimeLayout2), Msg: msg})
	}
	return dtos, nil
}
