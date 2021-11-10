package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func homepage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var member Member
	err = json.Unmarshal(reqBody, &member)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Print("Err Unmarshal")
		return
	}

	//insert member to database
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://behnamou:Behnam-2384@cluster0.u2qxk.mongodb.net/myFirstDatabase?retryWrites=true&w=majority"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = client.Connect(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)

	memberCollection := client.Database("test").Collection("Member")

	res, err := memberCollection.InsertOne(ctx, member)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	member.ID = res.InsertedID.(primitive.ObjectID)

	//response 200 to writer

	w.WriteHeader(http.StatusOK)

}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("v1/auth/register", homepage).Methods("POST")

	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func main() {
	handleRequests()
	fmt.Println("hello")
}

type Member struct {
	ID       primitive.ObjectID `bson:"id"`
	Username string
	Name     string
	Lastname string
	Mobile   string
	Email    string
	Password string
}
