package models

type User struct {
	Id       int
	UserName string
	Avatar   *string
}

type CategoryDto struct {
	Id   int
	Name string
}

type ArticleDto struct {
	Id           int
	Title        string
	Summary      string
	Html         string
	Content      string
	InsertTime   string
	ArticleType  int
	CategoryId   int
	UserId       int
	CategoryName string
	Cover        *string
}
type UserDto struct {
	Id             int
	UserName       string
	Level2pwd      *string
	EmailConfirmed int
	SecurityStamp  *string
	Email          *string
}

type WsmessageDto struct {
	InsertTime string
	Msg        string
}
