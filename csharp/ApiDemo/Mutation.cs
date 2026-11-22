public class Mutation
{
    public async Task<Post> CreatePost(
        string title,
        string content,
        int authorId,
        [Service] AppDbContext db)
    {
        var post = new Post
        {
            Title = title,
            Content = content,
            AuthorId = authorId
        };

        db.Posts.Add(post);
        await db.SaveChangesAsync();
        return post;
    }
}