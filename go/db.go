//структуры и работа с SQLite — модели данных и SQL-запросы

package main

import "database/sql"

type Author struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Post struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	AuthorID int    `json:"author_id"`
}

type Comment struct {
	ID       int    `json:"id"`
	PostID   int    `json:"post_id"`
	AuthorID int    `json:"author_id"`
	Text     string `json:"text"`
}

var db *sql.DB

func getAuthors() ([]Author, error) {
	rows, err := db.Query("SELECT id, name, age FROM authors")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	authors := []Author{}
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name, &a.Age); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}
	return authors, nil
}

func getAuthorByID(id int) (Author, error) {
	var a Author
	err := db.QueryRow("SELECT id, name, age FROM authors WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.Age)
	return a, err
}

func getPosts() ([]Post, error) {
	rows, err := db.Query("SELECT id, title, content, author_id FROM posts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func getPostByID(id int) (Post, error) {
	var p Post
	err := db.QueryRow("SELECT id, title, content, author_id FROM posts WHERE id = ?", id).
		Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID)
	return p, err
}

func getPostsByAuthorID(authorID int) ([]Post, error) {
	rows, err := db.Query("SELECT id, title, content, author_id FROM posts WHERE author_id = ?", authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.AuthorID); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func createPost(title, content string, authorID int) (Post, error) {
	res, err := db.Exec(
		"INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)",
		title, content, authorID,
	)
	if err != nil {
		return Post{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Post{}, err
	}
	return Post{ID: int(id), Title: title, Content: content, AuthorID: authorID}, nil
}

func getComments() ([]Comment, error) {
	rows, err := db.Query("SELECT id, post_id, author_id, text FROM comments")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.Text); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func getCommentsByPostID(postID int) ([]Comment, error) {
	rows, err := db.Query("SELECT id, post_id, author_id, text FROM comments WHERE post_id = ?", postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.Text); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func createComment(postID, authorID int, text string) (Comment, error) {
	res, err := db.Exec(
		"INSERT INTO comments (post_id, author_id, text) VALUES (?, ?, ?)",
		postID, authorID, text,
	)
	if err != nil {
		return Comment{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Comment{}, err
	}
	return Comment{ID: int(id), PostID: postID, AuthorID: authorID, Text: text}, nil
}

func deletePost(id int) (bool, error) {
	if _, err := db.Exec("DELETE FROM comments WHERE post_id = ?", id); err != nil {
		return false, err
	}

	res, err := db.Exec("DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
