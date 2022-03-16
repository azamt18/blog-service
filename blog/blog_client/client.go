package main

import (
	"blog/blog/blogpb"
	"context"
	"fmt"
	"google.golang.org/grpc"

	"log"
)

func main() {

	fmt.Println("Starting Blog Client")

	opts := grpc.WithInsecure()
	cc, err := grpc.Dial("localhost:50051", opts)
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer cc.Close()

	c := blogpb.NewBlogServiceClient(cc)

	// create blog
	fmt.Printf("Creating the blog...\n")
	blog := &blogpb.Blog{
		AuthorId: "Mark",
		Title:    "Title 1",
		Content:  "Content 1",
	}

	createResponse, createBlogError := c.CreateBlog(context.Background(), &blogpb.CreateBlogRequest{Blog: blog})
	if createBlogError != nil {
		log.Fatalf("Unexpected createBlogError: %v", createBlogError)
	}

	fmt.Printf("Blog has been created: %v\n", createResponse)
	blogId := createResponse.GetBlog().GetId()

	// read blog
	{
		fmt.Printf("Reading the blog...\n")

		_, readBlogError := c.ReadBlog(context.Background(), &blogpb.ReadBlogRequest{BlogId: "62318bb6d1dc923a9a5b11d0"})
		if readBlogError != nil {
			fmt.Printf("Error happened while reading: %v\n", readBlogError)
		}

		readBlogRequest := &blogpb.ReadBlogRequest{BlogId: blogId}
		readBlogResponse, readBlogError := c.ReadBlog(context.Background(), readBlogRequest)
		if readBlogError != nil {
			fmt.Printf("Error happened while reading: %v\n", readBlogError)
		}

		fmt.Printf("ReadBlog response: %v\n", readBlogResponse)
	}

	// update Blog
	{
		fmt.Printf("Updating the blog...\n")

		newBlog := &blogpb.Blog{
			Id:       blogId,
			AuthorId: "Author 2",
			Title:    "Title 2",
			Content:  "Content 2",
		}

		updateBlogResponse, updateBlogError := c.UpdateBlog(context.Background(), &blogpb.UpdateBlogRequest{Blog: newBlog})
		if updateBlogError != nil {
			fmt.Printf("Error happened while reading: %v\n", updateBlogError)
		}

		fmt.Printf("UpdateBlog response: %v\n", updateBlogResponse)
	}

	// delete Blog
	{
		deleteBlogResponse, deleteBlogError := c.DeleteBlog(context.Background(), &blogpb.DeleteBlogRequest{BlogId: blogId})
		if deleteBlogError != nil {
			fmt.Printf("Error while deleting: %v", deleteBlogError)
		}

		fmt.Printf("Blog was deleted: %v", deleteBlogResponse)
	}

}
