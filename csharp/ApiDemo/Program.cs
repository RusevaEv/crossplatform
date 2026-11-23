using Microsoft.EntityFrameworkCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseSqlite("Data Source=../../db/database.db"));

builder.Services
    .AddGraphQLServer()
    .AddQueryType<Query>()
    .AddMutationType<Mutation>();

var app = builder.Build();


// все авторы (без постов)
app.MapGet("/api/authors", async (AppDbContext db) =>
    await db.Authors.ToListAsync());

// конкретный автор (без постов)
app.MapGet("/api/authors/{id}", async (int id, AppDbContext db) =>
    await db.Authors.FindAsync(id) is Author author
        ? Results.Ok(author)
        : Results.NotFound());

// посты конкретного автора (без данных самого автора)
app.MapGet("/api/authors/{id}/posts", async (int id, AppDbContext db) =>
    await db.Posts.Where(p => p.AuthorId == id).ToListAsync());

// создание нового поста через REST
app.MapPost("/api/posts", async (Post post, AppDbContext db) =>
{
    db.Posts.Add(post);
    await db.SaveChangesAsync();
    return Results.Created($"/api/posts/{post.Id}", post);
});

app.MapGraphQL();

app.Run("http://localhost:5000");