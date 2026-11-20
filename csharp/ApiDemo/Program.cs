using Microsoft.EntityFrameworkCore;

var builder = WebApplication.CreateBuilder(args);

// Подключаемся к общей БД SQLite (путь относительно папки проекта ApiDemo)
builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseSqlite("Data Source=../../db/database.db"));

// Подключаем сервер GraphQL
builder.Services
    .AddGraphQLServer()
    .AddQueryType<Query>();

var app = builder.Build();

// --- ЧАСТЬ 1: REST API ---

// 1. Все авторы (без постов)
app.MapGet("/api/authors", async (AppDbContext db) =>
    await db.Authors.ToListAsync());

// 2. Конкретный автор (без постов)
app.MapGet("/api/authors/{id}", async (int id, AppDbContext db) =>
    await db.Authors.FindAsync(id) is Author author
        ? Results.Ok(author)
        : Results.NotFound());

// 3. Посты конкретного автора (без данных самого автора)
app.MapGet("/api/authors/{id}/posts", async (int id, AppDbContext db) =>
    await db.Posts.Where(p => p.AuthorId == id).ToListAsync());

// --- ЧАСТЬ 2: GRAPHQL ---
app.MapGraphQL(); // По умолчанию открывается на /graphql

// Запускаем строго на порту 5000, чтобы не пересекаться с Python и Go
app.Run("http://localhost:5000");