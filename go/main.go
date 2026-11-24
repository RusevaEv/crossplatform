//точка входа и запуск сервера — подключение к БД, регистрация маршрутов и запуск сервера

package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

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
	mux.HandleFunc("GET /api/posts/{id}", getPostByIDHandler)
	mux.HandleFunc("GET /api/posts/{id}/comments", getPostCommentsHandler)
	mux.HandleFunc("POST /api/posts", createPostHandler)
	mux.HandleFunc("GET /api/comments", getCommentsHandler)
	mux.HandleFunc("POST /api/comments", createCommentHandler)
	mux.HandleFunc("DELETE /api/posts/{id}", deletePostHandler)

	// GraphQL маршрут
	mux.HandleFunc("POST /graphql", makeGraphQLHandler(schema))

	fmt.Println("Go-сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
