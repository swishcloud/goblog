package models

import "time"

type CategoryDto struct {
	Id   int
	Name string
}

type ArticleDto struct {
	Id         int
	Title      string
	Summary    string
	Html       string
	Content    string
	InsertTime time.Time
	UpdateTime *time.Time
	//1 public, 2 private, 3 locked, 4 backup, 5 shared.
	ArticleType       int
	ShareDeadlineTime *time.Time
	CategoryId        int
	UserId            int
	UserName          string
	CategoryName      string
	Cover             *string
	ExpireTime        string
}
type UserDto struct {
	Id          int
	UserName    string
	Level2pwd   *string
	Insert_time time.Time
	Update_time *time.Time
	Is_banned   bool
	Op_issuer   string
	Op_userid   string
	Avatar      string
	Email       *string
}

type WsmessageDto struct {
	InsertTime string
	Msg        string
}

type FriendlyLink struct {
	Id                     string
	Description            string
	Website_url            string
	Friendly_link_page_url string
	Insert_time            time.Time
	Access_time            *time.Time
	Is_approved            bool
	Is_deleted             bool
	Website_name           string
}
