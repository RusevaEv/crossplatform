//схема и резолверы GraphQL — схема, резолверы и обработчик GraphQL

package main

import (
	"encoding/json"
	"net/http"

	"github.com/graphql-go/graphql"
)

func buildGraphQLSchema() (graphql.Schema, error) {
	authorType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Author",
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.Int},
			"name": &graphql.Field{Type: graphql.String},
			"age":  &graphql.Field{Type: graphql.Int},
		},
	})

	commentType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Comment",
		Fields: graphql.Fields{
			"id":       &graphql.Field{Type: graphql.Int},
			"postId":   &graphql.Field{Type: graphql.Int},
			"authorId": &graphql.Field{Type: graphql.Int},
			"text":     &graphql.Field{Type: graphql.String},
			"author": &graphql.Field{
				Type: authorType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					comment, ok := p.Source.(Comment)
					if !ok {
						return nil, nil
					}
					return getAuthorByID(comment.AuthorID)
				},
			},
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
			"comments": &graphql.Field{
				Type: graphql.NewList(commentType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					post, ok := p.Source.(Post)
					if !ok {
						return nil, nil
					}
					return getCommentsByPostID(post.ID)
				},
			},
		},
	})

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

	commentType.AddFieldConfig("post", &graphql.Field{
		Type: postType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			comment, ok := p.Source.(Comment)
			if !ok {
				return nil, nil
			}
			return getPostByID(comment.PostID)
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
			"post": &graphql.Field{
				Type: postType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getPostByID(p.Args["id"].(int))
				},
			},
			"comments": &graphql.Field{
				Type: graphql.NewList(commentType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getComments()
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
			"createComment": &graphql.Field{
				Type: commentType,
				Args: graphql.FieldConfigArgument{
					"postId":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"authorId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"text":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return createComment(
						p.Args["postId"].(int),
						p.Args["authorId"].(int),
						p.Args["text"].(string),
					)
				},
			},
			"deletePost": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return deletePost(p.Args["id"].(int))
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
