using Microsoft.EntityFrameworkCore;

public class Query
{
    // Получить всех авторов (с подгрузкой постов)
    public IQueryable<Author> GetAuthors([Service] AppDbContext db) =>
        db.Authors.Include(a => a.Posts);

    // Получить одного автора по ID
    public Author? GetAuthor(int id, [Service] AppDbContext db) =>
        db.Authors.Include(a => a.Posts).FirstOrDefault(a => a.Id == id);
}