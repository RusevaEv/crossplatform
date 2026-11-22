CREATE TABLE IF NOT EXISTS comments (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id   INTEGER NOT NULL REFERENCES posts(id),
    author_id INTEGER NOT NULL REFERENCES authors(id),
    text      TEXT NOT NULL
);

INSERT INTO comments (post_id, author_id, text) VALUES
(101, 2, 'Отличный разбор!'),
(101, 3, 'А есть пример с EF Core?'),
(102, 2, 'Как раз то, что нужно для проекта.');
