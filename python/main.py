from __future__ import annotations          

import os
import sqlite3
from contextlib import closing             
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import strawberry
from strawberry.fastapi import GraphQLRouter

DB_PATH = os.path.join(os.path.dirname(__file__), "..", "db", "database.db")


def get_db() -> sqlite3.Connection:
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys = ON")
    return conn


def db():                                    
    return closing(get_db())


# ---------- GraphQL-типы ----------

@strawberry.type
class PostType:
    id: int
    title: str
    content: str
    author_id: int

    @strawberry.field
    def comments(self) -> list[CommentType]:
        with db() as conn:
            rows = conn.execute(
                "SELECT id, post_id, author_id, text FROM comments WHERE post_id = ?",
                (self.id,),
            ).fetchall()
            return [CommentType(**dict(r)) for r in rows]


@strawberry.type
class CommentType:
    id: int
    post_id: int
    author_id: int
    text: str

    @strawberry.field
    def author(self) -> AuthorType:
        with db() as conn:
            row = conn.execute(
                "SELECT id, name, age FROM authors WHERE id = ?",
                (self.author_id,),
            ).fetchone()
            return AuthorType(**dict(row))


@strawberry.type
class AuthorType:
    id: int
    name: str
    age: int

    @strawberry.field
    def posts(self) -> list[PostType]:
        with db() as conn:
            rows = conn.execute(
                "SELECT id, title, content, author_id FROM posts WHERE author_id = ?",
                (self.id,),
            ).fetchall()
            return [PostType(**dict(r)) for r in rows]


# ---------- Query / Mutation ----------

@strawberry.type
class Query:
    @strawberry.field
    def authors(self) -> list[AuthorType]:
        with db() as conn:
            rows = conn.execute("SELECT id, name, age FROM authors").fetchall()
            return [AuthorType(**dict(r)) for r in rows]

    @strawberry.field
    def author(self, id: int) -> Optional[AuthorType]:
        with db() as conn:
            row = conn.execute(
                "SELECT id, name, age FROM authors WHERE id = ?", (id,)
            ).fetchone()
            return AuthorType(**dict(row)) if row else None

    @strawberry.field
    def posts(self) -> list[PostType]:
        with db() as conn:
            rows = conn.execute(
                "SELECT id, title, content, author_id FROM posts"
            ).fetchall()
            return [PostType(**dict(r)) for r in rows]

    @strawberry.field
    def post(self, id: int) -> Optional[PostType]:
        with db() as conn:
            row = conn.execute(
                "SELECT id, title, content, author_id FROM posts WHERE id = ?",
                (id,),
            ).fetchone()
            return PostType(**dict(row)) if row else None


@strawberry.type
class Mutation:
    @strawberry.mutation
    def create_post(self, title: str, content: str, author_id: int) -> PostType:
        with db() as conn:
            cur = conn.execute(
                "INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)",
                (title, content, author_id),
            )
            conn.commit()
            return PostType(
                id=cur.lastrowid, title=title,
                content=content, author_id=author_id,
            )

    @strawberry.mutation
    def create_comment(self, post_id: int, author_id: int, text: str) -> CommentType:
        with db() as conn:
            cur = conn.execute(
                "INSERT INTO comments (post_id, author_id, text) VALUES (?, ?, ?)",
                (post_id, author_id, text),
            )
            conn.commit()
            return CommentType(
                id=cur.lastrowid, post_id=post_id,
                author_id=author_id, text=text,
            )


schema = strawberry.Schema(query=Query, mutation=Mutation)
app = FastAPI()


# ---------- REST ----------

class PostCreate(BaseModel):
    title: str
    content: str
    authorId: int


class CommentCreate(BaseModel):
    postId: int
    authorId: int
    text: str


@app.get("/api/authors")
def get_authors():
    with db() as conn:
        return [dict(r) for r in
                conn.execute("SELECT id, name, age FROM authors").fetchall()]


@app.get("/api/authors/{author_id}")
def get_author(author_id: int):
    with db() as conn:
        row = conn.execute(
            "SELECT id, name, age FROM authors WHERE id = ?", (author_id,)
        ).fetchone()
        if not row:
            raise HTTPException(404, "Author not found")
        return dict(row)


@app.get("/api/authors/{author_id}/posts")
def get_author_posts(author_id: int):
    with db() as conn:
        rows = conn.execute(
            "SELECT id, title, content, author_id as authorId "
            "FROM posts WHERE author_id = ?",
            (author_id,),
        ).fetchall()
        return [dict(r) for r in rows]


@app.post("/api/posts", status_code=201)
def create_post_rest(post: PostCreate):
    with db() as conn:
        cur = conn.execute(
            "INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)",
            (post.title, post.content, post.authorId),
        )
        conn.commit()
        return {"id": cur.lastrowid, "title": post.title,
                "content": post.content, "authorId": post.authorId}


@app.get("/api/posts/{post_id}/comments")
def get_post_comments(post_id: int):
    with db() as conn:
        rows = conn.execute(
            "SELECT id, post_id as postId, author_id as authorId, text "
            "FROM comments WHERE post_id = ?",
            (post_id,),
        ).fetchall()
        return [dict(r) for r in rows]


@app.post("/api/comments", status_code=201)
def create_comment_rest(comment: CommentCreate):
    with db() as conn:
        cur = conn.execute(
            "INSERT INTO comments (post_id, author_id, text) VALUES (?, ?, ?)",
            (comment.postId, comment.authorId, comment.text),
        )
        conn.commit()
        return {"id": cur.lastrowid, "postId": comment.postId,
                "authorId": comment.authorId, "text": comment.text}


app.include_router(GraphQLRouter(schema), prefix="/graphql")