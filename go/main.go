package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/graphql-go/graphql"
	_ "modernc.org/sqlite"
)

// Модели данных (соответствуют таблицам в db/init.sql)
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

var db *sql.DB

// ================== РАБОТА С БАЗОЙ ДАННЫХ ==================

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
	return Post{
		ID:       int(id),
		Title:    title,
		Content:  content,
		AuthorID: authorID,
	}, nil
}

// ================== REST API ОБРАБОТЧИКИ ==================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getAuthorsHandler(w http.ResponseWriter, r *http.Request) {
	authors, err := getAuthors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, authors)
}

func getAuthorByIDHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	author, err := getAuthorByID(id)
	if err == sql.ErrNoRows {
		http.Error(w, "Автор не найден", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, author)
}

func getAuthorPostsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	posts, err := getPostsByAuthorID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := getPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		AuthorID int    `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	post, err := createPost(input.Title, input.Content, input.AuthorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

// ================== GRAPHQL СХЕМА И РЕЗОЛВЕРЫ ==================

func buildGraphQLSchema() (graphql.Schema, error) {
	authorType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Author",
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.Int},
			"name": &graphql.Field{Type: graphql.String},
			"age":  &graphql.Field{Type: graphql.Int},
		},
	})

	postType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Post",
		Fields: graphql.Fields{
			"id":       &graphql.Field{Type: graphql.Int},
			"title":    &graphql.Field{Type: graphql.String},
			"content":  &graphql.Field{Type: graphql.String},
			"authorId": &graphql.Field{Type: graphql.Int},
			"author": &graphql.Field{
				Type: authorType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					post, ok := p.Source.(Post)
					if !ok {
						return nil, nil
					}
					return getAuthorByID(post.AuthorID)
				},
			},
		},
	})

	// Добавляем автору поле posts после создания postType (чтобы избежать циклической ссылки)
	authorType.AddFieldConfig("posts", &graphql.Field{
		Type: graphql.NewList(postType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			author, ok := p.Source.(Author)
			if !ok {
				return nil, nil
			}
			return getPostsByAuthorID(author.ID)
		},
	})

	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"authors": &graphql.Field{
				Type: graphql.NewList(authorType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getAuthors()
				},
			},
			"author": &graphql.Field{
				Type: authorType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getAuthorByID(p.Args["id"].(int))
				},
			},
			"posts": &graphql.Field{
				Type: graphql.NewList(postType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getPosts()
				},
			},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createPost": &graphql.Field{
				Type: postType,
				Args: graphql.FieldConfigArgument{
					"title":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"content":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"authorId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return createPost(
						p.Args["title"].(string),
						p.Args["content"].(string),
						p.Args["authorId"].(int),
					)
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	})
}

func makeGraphQLHandler(schema graphql.Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Некорректный GraphQL-запрос", http.StatusBadRequest)
			return
		}

		result := graphql.Do(graphql.Params{
			Schema:         schema,
			RequestString:  req.Query,
			VariableValues: req.Variables,
		})
		writeJSON(w, http.StatusOK, result)
	}
}

// ================== ЗАПУСК СЕРВЕРА ==================

func main() {
	var err error
	db, err = sql.Open("sqlite", "../db/database.db")
	if err != nil {
		log.Fatal("Ошибка открытия БД:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Не удалось подключиться к БД:", err)
	}

	schema, err := buildGraphQLSchema()
	if err != nil {
		log.Fatal("Ошибка сборки GraphQL-схемы:", err)
	}

	mux := http.NewServeMux()

	// REST маршруты
	mux.HandleFunc("GET /api/authors", getAuthorsHandler)
	mux.HandleFunc("GET /api/authors/{id}", getAuthorByIDHandler)
	mux.HandleFunc("GET /api/authors/{id}/posts", getAuthorPostsHandler)
	mux.HandleFunc("GET /api/posts", getPostsHandler)
	mux.HandleFunc("POST /api/posts", createPostHandler)

	// GraphQL маршрут
	mux.HandleFunc("POST /graphql", makeGraphQLHandler(schema))

	fmt.Println("Go-сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
