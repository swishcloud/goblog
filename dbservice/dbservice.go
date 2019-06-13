package dbservice

import (
	"database/sql"
	"fmt"
	"github.com/github-123456/goblog/common"
	"github.com/github-123456/gostudy/aesencryption"
	"github.com/github-123456/gostudy/superdb"
	"strconv"
	"time"
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

func GetArticles(articleType, userId int, key string, withLockedContext bool) []*ArticleDto {
	var typeWhere string
	var userIdWhere string
	if articleType == 0 {
		typeWhere = ""
	} else {
		typeWhere = " and type=" + strconv.Itoa(articleType)
	}

	if userId == 0 {
		userIdWhere = ""
	} else {
		userIdWhere = " and userId=" + strconv.Itoa(userId)
	}
	rows, err := db.Query("select id,title,summary,content,insertTime,categoryId,type from article where title like ? "+typeWhere+userIdWhere+" and type!=4 and isDeleted=0 and isBanned=0 order by updateTime desc", "%"+key+"%")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	var articles []*ArticleDto
	for rows.Next() {
		var (
			id          int
			title       string
			summary     string
			content     string
			insertTime  string
			categoryId  int
			articleType int
		)
		if err := rows.Scan(&id, &title, &summary, &content, &insertTime, &categoryId, &articleType); err != nil {
			panic(err)
		}
		articles = append(articles, &ArticleDto{Id: id, Title: title, Summary: summary, Content: content, InsertTime: insertTime, CategoryId: categoryId, ArticleType: articleType})
	}
	for _, v := range articles {
		if v.ArticleType == 3 && !withLockedContext {
			v.Content = ""
		}
	}
	return articles
}

func GetCategories(userId int) []CategoryDto {
	rows, err := db.Query("select id,name from category where isdeleted=0 and userId=? order by name", userId)
	if err != nil {
		panic(err)
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

func UpdateCategory(name string, id, loginUserId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
		tx.Exec(`update category set name=? where id=? and userId=?`, name, id, loginUserId)
	}
}

func SetLevelTwoPwd(oldPwd string, pwd string, userId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
		tx.Exec("update user set level2pwd=? where id=?", aesencryption.Encrypt(pwd, time.Now().String()), userId)
		articles := GetArticles(3, userId, "", true)
		for _, v := range articles {
			cPlain, err := aesencryption.Decrypt(oldPwd, v.Content)
			if err != nil {
				panic(err)
			}
			cCipher := aesencryption.Encrypt(pwd, cPlain)
			tx.Exec(`update article set content=?,updateTime=? where id=?`, cCipher, time.Now(), v.Id)
		}
	}
}
func NewArticle(title string, summary string, html string, content string, userId int, articleType int, categoryId int, level2pwd string) superdb.DbTask {
	return func(tx *superdb.Tx) {
		if articleType == 3 {
			user := GetUser(userId)
			if user.Level2pwd == nil {
				//return dbServiceError{"您未设置二级密码"}
				panic("您未设置二级密码")
			}
			if !common.Lev2PwdCheck(*user.Level2pwd, level2pwd) {
				//return dbServiceError{"二级密码错误"}
				panic("二级密码错误")
			}
			content = aesencryption.Encrypt(level2pwd, content)
		}
		tx.MustExec(`insert into article (title,summary,html,content,userId,insertTime,updateTime,isDeleted,isBanned,type,categoryId)values(?,?,?,?,?,?,?,?,?,?,?)`, title, summary, html, content, userId, time.Now(), time.Now(), 0, 0, articleType, categoryId)
	}
}
func UpdateArticle(id int, title string, summary string, html string, content string, articleType int, categoryId, level2pwd string, userId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
		if articleType == 3 {
			user := GetUser(userId)
			if user.Level2pwd == nil {
				//return dbServiceError{"您未设置二级密码"}
				panic("您未设置二级密码")
			}
			if !common.Lev2PwdCheck(*user.Level2pwd, level2pwd) {
				//return dbServiceError{"二级密码错误"}
				panic("二级密码错误")
			}
			content = aesencryption.Encrypt(level2pwd, content)
		} else if articleType != 1 && articleType != 2 {
			panic(fmt.Sprintf("articleType %d is invalid", articleType))
		}
		tx.MustExec(`update article set title=?,summary=?,html=?,content=?,type=?,categoryId=?,updateTime=? where id=?`, title, summary, html, content, articleType, categoryId, time.Now(), id)
	}
}

func GetArticle(id int) *ArticleDto {
	r := db.QueryRow("select id,title,html,content,insertTime,type,categoryId,userId from article where id=?", id)
	var (
		title       string
		html        string
		content     string
		insertTime  string
		articleType int
		categoryId  int
		userId      int
	)
	if err := r.Scan(&id, &title, &html, &content, &insertTime, &articleType, &categoryId, &userId); err != nil {
		return nil
	}
	return &ArticleDto{Title: title, Html: html, Content: content, InsertTime: insertTime, Id: id, ArticleType: articleType, CategoryId: categoryId, UserId: userId}
}

func ArticleDelete(id, loginUserId int)  superdb.DbTask  {
	return func(tx *superdb.Tx) {

		var article = GetArticle(id)
		if article.UserId != loginUserId {
			panic("no permission")
		}
		tx.MustExec("update article set isDeleted=1 where id=?",id)
	}
}

func GetUser(userId int) *UserDto {
	r := db.QueryRow("select id,userName,level2pwd from user where id=? and isdeleted=0 and isBanned=0", userId)
	var (
		id        int
		userName  string
		level2pwd *string
	)
	if err := r.Scan(&id, &userName, &level2pwd); err != nil {
		return nil
	}
	return &UserDto{Id: userId, UserName: userName, Level2pwd: level2pwd}
}

func NewCategory(name string, userId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
		tx.MustExec(`insert into category (name,insertTime,isDeleted,userId)values(?,?,?,?)`, name, time.Now(), 0, userId)
	}
}

func NewUser(userName, password string) superdb.DbTask {
	return func(tx *superdb.Tx) {
		r := tx.MustExec(`insert into user (userName,password,insertTime,isDeleted,isBanned)values(?,?,?,?,?)`, userName, common.Md5Hash(password), time.Now(), 0, 0)
		lastId, err := r.LastInsertId()
		if err != nil {
			panic(err)
		}
		intLastId := int(lastId)
		NewCategory("默认分类", intLastId)(tx)
	}
}
