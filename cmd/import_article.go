package cmd

import (
	"github.com/spf13/cobra"
)

var importArticleCmd = &cobra.Command{
	Use: "article",
	Run: func(cmd *cobra.Command, args []string) {
		/*source_path, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}
		conn_info, err := cmd.Flags().GetString("conn_info")
		if err != nil {
			log.Fatal(err)
		}
		key, err := cmd.Flags().GetString("key")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("start importing articles from", source_path)
		b, err := ioutil.ReadFile(source_path)
		if err != nil {
			log.Fatal(err)
		}
		data := []map[string]interface{}{}
		json.Unmarshal(b, &data)
		storage := storage.NewSQLManager(conn_info)
		for i := 0; i < len(data); i++ {
			article := data[i]
			articleType := article["type"].(float64)
			title := article["title"].(string)
			content := article["content"].(string)
			html := article["html"].(string)
			summary := article["summary"].(string)
			insertTime := article["insertTime"].(string)
			updateTime := article["updateTime"].(string)
			var cover *string = nil
			if c, ok := article["cover"].(string); ok {
				cover = &c
			}
			userId := article["userId"].(float64)
			categoryId := 0
			if userId == 12 {
				categoryId = 6
			} else if userId == 109 {
				categoryId = 7
			} else {
				log.Fatal("can not import data to this user")
			}

			insert_t, err := time.Parse(internal.TimeLayoutMysqlDateTime, insertTime)
			insert_t = insert_t.UTC()
			if err != nil {
				log.Fatal(err)
			}
			update_t, err := time.Parse(internal.TimeLayoutMysqlDateTime, updateTime)
			update_t = update_t.UTC()
			if err != nil {
				log.Fatal(err)
			}
			storage.NewArticle(title, summary, html, content, int(userId), int(articleType), categoryId, key, cover, nil, &insert_t, &update_t, "import")

		}
		storage.Commit()
		println("finished importing articles")*/
	},
}

func init() {
	importCmd.AddCommand(importArticleCmd)
	importArticleCmd.Flags().String("file", "", "data source file path")
	importArticleCmd.Flags().String("conn_info", "", "database connection string")
	importArticleCmd.Flags().String("key", "", "post secret key")
	importArticleCmd.MarkFlagRequired("file")
	importArticleCmd.MarkFlagRequired("conn_info")
	importArticleCmd.MarkFlagRequired("key")
}
