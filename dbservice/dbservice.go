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

func GetArticles(articleType, userId int, key string, withLockedContext bool, categoryName string) []*ArticleDto {
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
		r, err := db.Query("select a.id,a.title,a.summary,a.html,a.content,a.insertTime,a.categoryId,a.userId,a.type,b.name as categoryName from article as a join category as b on a.categoryId=b.id where title like ? "+typeWhere+userIdWhere+" and type!=4 and a.isDeleted=0 and isBanned=0 order by a.updateTime desc", "%"+key+"%")
		if err != nil {
			panic(err.Error())
		}
		rows = r
	} else {
		r, err := db.Query("select a.id,a.title,a.summary,a.html,a.content,a.insertTime,a.categoryId,a.userId,a.type,b.name as categoryName from article as a join category as b on a.categoryId=b.id where b.name=? and title like ? "+typeWhere+userIdWhere+" and type!=4 and a.isDeleted=0 and isBanned=0 order by  a.updateTime desc", categoryName, "%"+key+"%")
		if err != nil {
			panic(err.Error())
		}
		rows = r
	}
	defer rows.Close()

	var articles []*ArticleDto
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
		)
		if err := rows.Scan(&id, &title, &summary, &html, &content, &insertTime, &categoryId, &userId, &articleType, &categoryName); err != nil {
			panic(err)
		}
		articles = append(articles, &ArticleDto{Id: id, Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, CategoryId: categoryId, UserId: userId, ArticleType: articleType, CategoryName: categoryName})
	}
	for _, v := range articles {
		if v.ArticleType == 3 && !withLockedContext {
			v.Content = ""
			v.Summary = ""
		}
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
func NewArticle(title string, summary string, html string, content string, userId int, articleType int, categoryId int, key string) superdb.DbTask {
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
		id,err:=tx.MustExec(`insert into article (title,summary,html,content,userId,insertTime,updateTime,isDeleted,isBanned,type,categoryId)values(?,?,?,?,?,?,?,?,?,?,?)`, title, summary, html, content, userId, time.Now(), time.Now(), 0, 0, articleType, categoryId).LastInsertId()
		if err!=nil{
			panic(err)
		}
		tx.SetValue("NewArticleLastInsertId",id)
	}
}
func UpdateArticle(id int, title string, summary string, html string, content string, articleType int, categoryId, key string, userId int) superdb.DbTask {
	return func(tx *superdb.Tx) {
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
		tx.MustExec(`update article set title=?,summary=?,html=?,content=?,type=?,categoryId=?,updateTime=? where id=?`, title, summary, html, content, articleType, categoryId, time.Now(), id)
	}
}

func GetArticle(id int) *ArticleDto {
	r := db.QueryRow("select id,title,summary,html,content,insertTime,type,categoryId,userId from article where id=?", id)
	var (
		title       string
		summary     string
		html        string
		content     string
		insertTime  string
		articleType int
		categoryId  int
		userId      int
	)
	if err := r.Scan(&id, &title, &summary, &html, &content, &insertTime, &articleType, &categoryId, &userId); err != nil {
		return nil
	}
	return &ArticleDto{Title: title, Summary: summary, Html: html, Content: content, InsertTime: insertTime, Id: id, ArticleType: articleType, CategoryId: categoryId, UserId: userId}
}

func ArticleDelete(id, loginUserId int) superdb.DbTask {
	return func(tx *superdb.Tx) {

		var article = GetArticle(id)
		if article.UserId != loginUserId {
			panic("no permission")
		}
		tx.MustExec("update article set isDeleted=1 where id=?", id)
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

func CheckUser(account, pwd string, maxAllowAccessFaildCount int) (int, error) {
	var (
		id                int
		password          string
		userName          string
		accessFailedCount int
		lockoutEnd        *string
	)
	r := db.QueryRow("select id,userName, password,accessFailedCount,lockoutEnd from user where userName=?", account)
	err := r.Scan(&id, &userName, &password, &accessFailedCount, &lockoutEnd)
	if err != nil {
		return id, common.Error{fmt.Sprintf("账号不存在")}
	}
	if lockoutEnd != nil {
		lockoutEndTime, err := time.Parse("2006-01-02 15:04:05", *lockoutEnd)
		if err != nil {
			return id, err
		}
		fmt.Println(lockoutEndTime.UTC(), time.Now().UTC())
		if lockoutEndTime.UTC().Before(time.Now().UTC()) {
			UnlockUser(id)
		} else {
			return id, common.Error{"您的账号已被锁定"}
		}
	}
	if !common.Md5Check(password, pwd) {
		failedCount := accessFailedCount + 1
		db.Exec("update user set accessFailedCount=? where userName=?", failedCount, account)
		if remainC := maxAllowAccessFaildCount - failedCount; remainC > 0 {
			return id, common.Error{fmt.Sprintf("密码错误，您还有%d次重试机会", remainC)}
		} else {
			LockUser(id)
			ResetAccessFailedCount(id)
			return id, common.Error{"您的账号已被锁定"}
		}
	}
	ResetAccessFailedCount(id)
	return id, nil
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
