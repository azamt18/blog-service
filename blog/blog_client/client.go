package main

import (
	"blog/blog/blogpb"
	"context"
	"fmt"
	"github.com/simplesteph/grpc-go-course/greet/greetpb"
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

	createBlogResponse, createBlogError := c.CreateBlog(context.Background(), &blogpb.CreateBlogRequest{Blog: blog})
	if createBlogError != nil {
		log.Fatalf("Unexpected createBlogError: %v", createBlogError)
	}

	fmt.Printf("Blog has been created: %v\n", createBlogResponse)

	// read blog
	fmt.Printf("Reading the blog...\n")

	readBlogResponse, readBlogError := c.ReadBlog(context.Background(), &blogpb.ReadBlogRequest{BlogId: "62318bb6d1dc923a9a5b11d0"})
	if readBlogError != nil {
		fmt.Printf("Error happened while reading: %v\n", readBlogError)
	}

	fmt.Printf("ReadBlog response: %v\n", readBlogResponse)

}

func doUnary(c greetpb.GreetServiceClient) {
	fmt.Println("Starting to do a Unary RPC...")
	req := &greetpb.GreetRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Stephane",
			LastName:  "Maarek",
		},
	}
	res, err := c.Greet(context.Background(), req)
	if err != nil {
		log.Fatalf("error while calling Greet RPC: %v", err)
	}
	log.Printf("Response from Greet: %v", res.Result)
}
