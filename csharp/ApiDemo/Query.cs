using Microsoft.EntityFrameworkCore;

public class Query
{
    public IQueryable<Author> GetAuthors([Service] AppDbContext db) =>
        db.Authors.Include(a => a.Posts);

    public Author? GetAuthor(int id, [Service] AppDbContext db) =>
        db.Authors.Include(a => a.Posts).FirstOrDefault(a => a.Id == id);
}