import os
import sqlite3
from typing import Optional
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import strawberry
from strawberry.fastapi import GraphQLRouter

DB_PATH = os.path.join(os.path.dirname(__file__), "..", "db", "database.db")

def get_db():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row  
    return conn

@strawberry.type
class PostType:
    id: int
    title: str
    content: str
    author_id: int  

@strawberry.type
class AuthorType:
    id: int
    name: str
    age: int
    @strawberry.field
    def posts(self) -> list[PostType]:
        with get_db() as conn:
            rows = conn.execute(
                "SELECT id, title, content, author_id FROM posts WHERE author_id = ?",
                (self.id,)
            ).fetchall()
            return [PostType(**dict(row)) for row in rows]

@strawberry.type
class Query:
    @strawberry.field
    def authors(self) -> list[AuthorType]:
        with get_db() as conn:
            rows = conn.execute("SELECT id, name, age FROM authors").fetchall()
            return [AuthorType(**dict(row)) for row in rows]

    @strawberry.field
    def author(self, id: int) -> Optional[AuthorType]:
        with get_db() as conn:
            row = conn.execute(
                "SELECT id, name, age FROM authors WHERE id = ?", (id,)
            ).fetchone()
            return AuthorType(**dict(row)) if row else None

@strawberry.type
class Mutation:
    @strawberry.mutation
    def create_post(self, title: str, content: str, author_id: int) -> PostType:
        with get_db() as conn:
            cursor = conn.execute(
                "INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)",
                (title, content, author_id)
            )
            conn.commit()
            return PostType(
                id=cursor.lastrowid,
                title=title,
                content=content,
                author_id=author_id
            )

schema = strawberry.Schema(query=Query, mutation=Mutation)


app = FastAPI()

class PostCreate(BaseModel):
    title: str
    content: str
    authorId: int

@app.get("/api/authors")
def get_authors():
    with get_db() as conn:
        rows = conn.execute("SELECT id, name, age FROM authors").fetchall()
        return [dict(row) for row in rows]

@app.get("/api/authors/{author_id}")
def get_author(author_id: int):
    with get_db() as conn:
        row = conn.execute(
            "SELECT id, name, age FROM authors WHERE id = ?", (author_id,)
        ).fetchone()
        if not row:
            raise HTTPException(status_code=404, detail="Author not found")
        return dict(row)

@app.get("/api/authors/{author_id}/posts")
def get_author_posts(author_id: int):
    with get_db() as conn:
        rows = conn.execute(
            "SELECT id, title, content, author_id as authorId FROM posts WHERE author_id = ?",
            (author_id,)
        ).fetchall()
        return [dict(row) for row in rows]

@app.post("/api/posts", status_code=201)
def create_post_rest(post: PostCreate):
    with get_db() as conn:
        cursor = conn.execute(
            "INSERT INTO posts (title, content, author_id) VALUES (?, ?, ?)",
            (post.title, post.content, post.authorId)
        )
        conn.commit()
        return {
            "id": cursor.lastrowid,
            "title": post.title,
            "content": post.content,
            "authorId": post.authorId
        }

app.include_router(GraphQLRouter(schema), prefix="/graphql")