package dbservice

import (
	"database/sql"
	"fmt"
	"github.com/github-123456/goblog/common"
	"github.com/github-123456/gostudy/aesencryption"
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

func GetArticles(articleType, userId int, key string) []*ArticleDto {
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
	rows, err := db.Query("select id,title,content,insertTime,categoryId,type from article where title like ? "+typeWhere+userIdWhere+" order by updateTime desc", "%"+key+"%")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	var articles []*ArticleDto
	for rows.Next() {
		var (
			id         int
			title      string
			content    string
			insertTime string
			categoryId int
			articleType int
		)
		if err := rows.Scan(&id, &title, &content, &insertTime,&categoryId,&articleType); err != nil {
			panic(err)
		}
		articles = append(articles, &ArticleDto{Id: id, Title: title, Content: content, InsertTime: insertTime,CategoryId:categoryId,ArticleType:articleType})
	}
	for _,v:=range articles{
		if v.ArticleType==3{
			v.Content=""
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

func SetLevelTwoPwd(pwd string, userId int) {
	_, err := db.Exec("update user set level2pwd=? where id=?", common.HashPwd(pwd), userId)
	if err != nil {
		panic(err)
	}
}
func NewArticle(title string, content string, userId int, articleType int, categoryId int, level2pwd string)  {
	if articleType == 3 {
		user := GetUser(userId)
		if user.Level2pwd == nil {
			//return dbServiceError{"用户未设置二级密码"}
			panic("用户未设置二级密码")
		}
		if !common.PwdCheck(*user.Level2pwd, level2pwd) {
			//return dbServiceError{"二级密码错误"}
			panic("二级密码错误")
		}
		content = aesencryption.Encrypt(level2pwd, content)
	}

	_, err := db.Exec(`insert into article (title,content,userId,insertTime,updateTime,isDeleted,isBanned,type,categoryId)values(
	?,?,?,?,?,?,?,?,?
	)`, title, content, userId, time.Now(), time.Now(), 0, 0, articleType, categoryId)
	if err != nil {
		panic(err)
	}
}
func UpdateArticle(id int, title string, content string, articleType int, categoryId, level2pwd string,userId int) {
	if articleType == 3 {
		user := GetUser(userId)
		if user.Level2pwd == nil {
			//return dbServiceError{"用户未设置二级密码"}
			panic("用户未设置二级密码")
		}
		if !common.PwdCheck(*user.Level2pwd, level2pwd) {
			//return dbServiceError{"二级密码错误"}
			panic("二级密码错误")
		}
		content = aesencryption.Encrypt(level2pwd, content)
	}else if articleType != 1 && articleType != 2 {
		panic(fmt.Sprintf("articleType %d is invalid", articleType))
	}
	_, err := db.Exec(`update article set title=?,content=?,type=?,categoryId=?,updateTime=? where id=?`, title, content, articleType, categoryId, time.Now(), id)
	if err != nil {
		panic(err)
	}
}

func GetArticle(id int) *ArticleDto {
	r := db.QueryRow("select id,title,content,insertTime,type,categoryId from article where id=?", id)
	var (
		title      string
		content    string
		insertTime string
		articleType   int
		categoryId int
	)
	if err := r.Scan(&id, &title, &content, &insertTime, &articleType, &categoryId); err != nil {
		return nil
	}
	return &ArticleDto{Title: title, Content: content, InsertTime: insertTime, Id: id, ArticleType: articleType, CategoryId: categoryId}
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
