package dbservice
type CategoryDto struct {
	Id   int
	Name string
}

type ArticleDto struct {
	Id         int
	Title      string
	Content    string
	InsertTime string
	ArticleType int
	CategoryId int
}
type UserDto struct{
	Id int
	UserName string
	Level2pwd *string
}