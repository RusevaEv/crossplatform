CREATE TABLE IF NOT EXISTS authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    age INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    FOREIGN KEY (author_id) REFERENCES authors(id)
);

INSERT INTO authors (id, name, age) VALUES 
(1, 'Анна Смирнова', 21),
(2, 'Иван Петров', 23),
(3, 'Мария Соколова', 20);

INSERT INTO posts (id, title, content, author_id) VALUES 
(101, 'Основы C# и .NET', 'Подробный разбор Minimal API...', 1),
(102, 'GraphQL на практике', 'Как избавиться от over-fetching...', 1),
(103, 'Асинхронность в Python', 'Разбираемся с FastAPI и uvicorn...', 2),
(104, 'Микросервисы на Go', 'Почему Go так популярен в бэкенде...', 3);