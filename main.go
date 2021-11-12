package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"net/smtp"

	"github.com/gorilla/mux"
	"github.com/kavenegar/kavenegar-go"
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

	//	send sms and email

	go sendSMS(member.Mobile)
	go sendEmail(member.Email)

	// ------------------------

	member.ID = res.InsertedID.(primitive.ObjectID)

	//response 200 to writer

	w.WriteHeader(http.StatusOK)

}

func sendEmail(toEmail string) {
	from := "bihnam998@gmail.com"
	pass := "bnegrwofqcdanthh"
	to := toEmail

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Registration\n\n" +
		"Register Successful!!!"

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("smtp error: %s", err)
		return
	}

	log.Print("sent, Done")
}

func sendSMS(sendNumber string) {
	api := kavenegar.New("663171736B65374D61476E67475957743543326F474F3254777943347469675063307845302F54384736673D")
	sender := "10008663"
	receptor := []string{sendNumber}
	message := "Register Successful!!!"
	if res, err := api.Message.Send(sender, receptor, message, nil); err != nil {
		switch err := err.(type) {
		case *kavenegar.APIError:
			fmt.Println(err.Error())
		case *kavenegar.HTTPError:
			fmt.Println(err.Error())
		default:
			fmt.Println(err.Error())
		}
	} else {
		for _, r := range res {
			fmt.Println("MessageID 	= ", r.MessageID)
			fmt.Println("Status    	= ", r.Status)
		}
	}
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/v1/auth/register", homepage).Methods(http.MethodPost)

	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func main() {
	handleRequests()
	fmt.Println("hello")
}

type Member struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `validate:"string,min=2,max=20"`
	Name     string             `validate:"string,min=2,max=20"`
	Lastname string             `validate:"string,min=2,max=20"`
	Mobile   string
	Email    string `validate:"email"`
	Password string `validate:"string,min=8,max=50"`
}

const tagName = "validate"

// add api validaton
//	send sms when done
//	send email when done
