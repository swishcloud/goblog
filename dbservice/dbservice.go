package dbservice

import (
	"database/sql"
)

var db *sql.DB

func SetDb(d *sql.DB) {
	db = d
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
