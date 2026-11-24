from __future__ import annotations          

import os
import sqlite3
from contextlib import closing             
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import strawberry
from strawberry.fastapi import GraphQLRouter

import asyncio
from typing import AsyncGenerator

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


# ---------- Query ----------

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

post_queue = asyncio.Queue()
comment_queue = asyncio.Queue()

@strawberry.type
class Subscription:
    @strawberry.subscription
    async def post_created(self) -> AsyncGenerator[PostType, None]:
        while True:
            new_post = await post_queue.get()
            yield new_post

    @strawberry.subscription
    async def comment_created(self) -> AsyncGenerator[CommentType, None]:
        while True:
            new_comment = await comment_queue.get()
            yield new_comment
# ---------- Mutation ----------

@strawberry.type
class Mutation:
    @strawberry.mutation
    async def create_post(self, title: str, content: str, author_id: int) -> PostType:
        with db() as conn:
            cur = conn.execute(
                "INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)",
                (title, content, author_id),
            )
            conn.commit()
            post = PostType(
                id=cur.lastrowid, title=title,
                content=content, author_id=author_id,
            )
        await post_queue.put(post)
        return post

    @strawberry.mutation
    async def create_comment(self, post_id: int, author_id: int, text: str) -> CommentType:
        with db() as conn:
            cur = conn.execute(
                "INSERT INTO comments (post_id, author_id, text) VALUES (?, ?, ?)",
                (post_id, author_id, text),
            )
            conn.commit()
            comment = CommentType(
                id=cur.lastrowid, post_id=post_id,
                author_id=author_id, text=text,
            )
        await comment_queue.put(comment)
        return comment

    @strawberry.mutation
    def update_post(self, id: int, title: str, content: str) -> Optional[PostType]:
        with db() as conn:
            exists = conn.execute("SELECT author_id FROM posts WHERE id = ?", (id,)).fetchone()
            if not exists:
                return None
                
            conn.execute(
                "UPDATE posts SET title = ?, content = ? WHERE id = ?",
                (title, content, id),
            )
            conn.commit()
 
            return PostType(id=id, title=title, content=content, author_id=exists["author_id"])

    @strawberry.mutation
    def update_comment(self, id: int, text: str) -> Optional[CommentType]:
        with db() as conn:
            exists = conn.execute("SELECT post_id, author_id FROM comments WHERE id = ?", (id,)).fetchone()
            if not exists:
                return None
                
            conn.execute("UPDATE comments SET text = ? WHERE id = ?", (text, id))
            conn.commit()
            return CommentType(id=id, post_id=exists["post_id"], author_id=exists["author_id"], text=text)

    @strawberry.mutation
    def update_author(self, id: int, name: str, age: int) -> Optional[AuthorType]:
        with db() as conn:
            exists = conn.execute("SELECT id FROM authors WHERE id = ?", (id,)).fetchone()
            if not exists:
                return None
                
            conn.execute("UPDATE authors SET name = ?, age = ? WHERE id = ?", (name, age, id))
            conn.commit()
            return AuthorType(id=id, name=name, age=age)


schema = strawberry.Schema(query=Query, mutation=Mutation, subscription=Subscription)
app = FastAPI()
app.include_router(GraphQLRouter(schema), prefix="/graphql")

# ---------- REST ----------

class PostCreate(BaseModel):
    title: str
    content: str
    authorId: int

class CommentCreate(BaseModel):
    postId: int
    authorId: int
    text: str

class PostUpdate(BaseModel):
    title: str
    content: str

class CommentUpdate(BaseModel):
    text: str

class AuthorUpdate(BaseModel):
    name: str
    age: int


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

@app.put("/api/posts/{post_id}")
def update_post_rest(post_id: int, post: PostUpdate):
    with db() as conn:
        exists = conn.execute("SELECT author_id FROM posts WHERE id = ?", (post_id,)).fetchone()
        if not exists:
            raise HTTPException(404, "Post not found")
            
        conn.execute(
            "UPDATE posts SET title = ?, content = ? WHERE id = ?",
            (post.title, post.content, post_id),
        )
        conn.commit()
        return {
            "id": post_id, 
            "title": post.title,
            "content": post.content, 
            "authorId": exists["author_id"]
        }

@app.put("/api/comments/{comment_id}")
def update_comment_rest(comment_id: int, comment: CommentUpdate):
    with db() as conn:
        exists = conn.execute("SELECT post_id, author_id FROM comments WHERE id = ?", (comment_id,)).fetchone()
        if not exists:
            raise HTTPException(404, "Comment not found")
            
        conn.execute("UPDATE comments SET text = ? WHERE id = ?", (comment.text, comment_id))
        conn.commit()
        return {
            "id": comment_id, 
            "postId": exists["post_id"],
            "authorId": exists["author_id"], 
            "text": comment.text
        }

@app.put("/api/authors/{author_id}")
def update_author_rest(author_id: int, author: AuthorUpdate):
    with db() as conn:
        exists = conn.execute("SELECT id FROM authors WHERE id = ?", (author_id,)).fetchone()
        if not exists:
            raise HTTPException(404, "Author not found")
            
        conn.execute("UPDATE authors SET name = ?, age = ? WHERE id = ?", (author.name, author.age, author_id))
        conn.commit()
        return {
            "id": author_id, 
            "name": author.name, 
            "age": author.age
        }
