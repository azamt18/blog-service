package main

import (
	"context"
	"fmt"
	"github.com/simplesteph/grpc-go-course/blog/blogpb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"os/signal"
)

var collection *mongo.Collection

type server struct{}

type blogItem struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id"`
	Content  string             `bson:"content"`
	Title    string             `bson:"title"`
}

func (s server) CreateBlog(ctx context.Context, request *blogpb.CreateBlogRequest) (*blogpb.CreateBlogResponse, error) {
	fmt.Println("Create blog request...")

	// create blog request
	blog := request.GetBlog()
	data := blogItem{
		AuthorID: blog.GetAuthorId(),
		Content:  blog.GetContent(),
		Title:    blog.GetTitle(),
	}

	result, error := collection.InsertOne(context.Background(), data)
	if error != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", error),
		)
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot convert to OID"),
		)
	}

	response := &blogpb.CreateBlogResponse{
		Blog: &blogpb.Blog{
			Id:       oid.Hex(),
			AuthorId: blog.GetAuthorId(),
			Title:    blog.GetTitle(),
			Content:  blog.GetContent(),
		},
	}

	return response, nil
}

func (s server) ReadBlog(ctx context.Context, request *blogpb.ReadBlogRequest) (*blogpb.ReadBlogResponse, error) {
	fmt.Println("Read blog request...")

	blogId := request.GetBlogId()
	oid, error := primitive.ObjectIDFromHex(blogId)
	if error != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	// create an empty struct
	data := &blogItem{}
	filter := bson.M{"_id": oid} // NewDocument

	// perform find operation
	result := collection.FindOne(ctx, filter)
	if error := result.Decode(data); error != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v", error),
		)
	}

	// prepare response
	response := &blogpb.ReadBlogResponse{
		Blog: dataToBlogPb(data),
	}

	return response, nil
}

func dataToBlogPb(data *blogItem) *blogpb.Blog {
	return &blogpb.Blog{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorID,
		Title:    data.Title,
		Content:  data.Content,
	}
}

func (s server) UpdateBlog(ctx context.Context, request *blogpb.UpdateBlogRequest) (*blogpb.UpdateBlogResponse, error) {
	fmt.Println("Update blog request...")
	blog := request.GetBlog()
	oid, error := primitive.ObjectIDFromHex(blog.GetId())
	if error != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	// create an empty struct
	data := &blogItem{}
	filter := bson.M{"_id": oid}

	result := collection.FindOne(ctx, filter)
	if error := result.Decode(data); error != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Can not find a blog with given ID: %v", error),
		)
	}

	// perform update operation
	data.AuthorID = blog.GetAuthorId()
	data.Title = blog.GetTitle()
	data.Content = blog.GetContent()

	_, updateError := collection.ReplaceOne(context.Background(), filter, data)
	if updateError != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Can not update object in the Db: %v", updateError),
		)
	}

	// prepare response
	response := &blogpb.UpdateBlogResponse{
		Blog: dataToBlogPb(data),
	}

	return response, nil
}

func (s server) DeleteBlog(ctx context.Context, request *blogpb.DeleteBlogRequest) (*blogpb.DeleteBlogResponse, error) {
	fmt.Println("Delete blog request...")
	oid, error := primitive.ObjectIDFromHex(request.GetBlogId())
	if error != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	filter := bson.M{"_id": oid}
	deleteResult, deleteError := collection.DeleteOne(context.Background(), filter)
	if deleteError != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Can not delete object in the Db: %v", deleteError),
		)
	}

	if deleteResult.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Can not find blog in the Db: %v", deleteError),
		)
	}

	return &blogpb.DeleteBlogResponse{
		BlogId: request.GetBlogId(),
	}, nil
}

func (s server) ListBlog(request *blogpb.ListBlogRequest, stream blogpb.BlogService_ListBlogServer) error {
	fmt.Println("List blog request...")

	cursor, error := collection.Find(context.Background(), primitive.D{{}}) // D - used because of the order of the elements matters
	if error != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", error),
		)
	}

	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			fmt.Printf("Error while closing a cursor: %v", err)
		}
	}(cursor, context.Background())

	for cursor.Next(context.Background()) {
		// create an empty struct for response
		data := &blogItem{}
		if error := cursor.Decode(data); error != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", error),
			)
		}

		// send a blog via stream
		if error := stream.Send(&blogpb.ListBlogResponse{Blog: dataToBlogPb(data)}); error != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while sending a stream: %v", error),
			)
		}
	}

	if err := cursor.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", error),
		)
	}

	return nil
}

func main() {
	// if the go code is crushed -> get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Connection to MongoDb")
	// Create client
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	// Create connect
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database("mydb").Collection("blog")

	fmt.Println("Start Blog Service...")
	lis, err := net.Listen("tcp", "0.0.0.0:50051")

	if err != nil {
		log.Fatalf("FAILED TO LISTEN %v", err)
	}

	var options []grpc.ServerOption
	s := grpc.NewServer(options...)
	blogpb.RegisterBlogServiceServer(s, &server{})

	// Register reflection service on gRPC server
	reflection.Register(s)

	go func() {
		fmt.Println("Starting Blog Server...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait for Control C to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block until a signal is received
	<-ch

	// 1st: close the connection with db
	fmt.Println("Closing MongoDb connection")
	client.Disconnect(context.TODO())

	// 2nd: close the listener
	fmt.Println("Closing the listener")
	lis.Close()

	// Finally, stop the server
	fmt.Println("Stopping the server")
	s.Stop()

	fmt.Println("End of Program")
}
