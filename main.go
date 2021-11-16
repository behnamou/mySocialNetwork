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

	"github.com/go-playground/validator/v10"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kavenegar/kavenegar-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var cache redis.Conn

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

	//	validation

	validate := validator.New()

	err = validate.Struct(member)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//	------------------

	// set password to hash

	newPass, err := HashPassword(member.Password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	member.Password = newPass

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
	fmt.Println("homepage done")

}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
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
	fmt.Println("email sent")
}

func sendSMS(sendNumber string) {
	time.Sleep(time.Duration(time.Second * 10))
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
	fmt.Println("sms sent")
	// fmt.Println("sms before duration")

}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/v1/auth/register", homepage).Methods(http.MethodPost)

	myRouter.HandleFunc("/login", signin).Methods(http.MethodPost)

	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func signin(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var logedInMember Member

	enteredPassword, err := HashPassword(creds.Password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// 	coonect database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	if err = memberCollection.FindOne(ctx, bson.M{}).Decode(&logedInMember); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if enteredPassword != logedInMember.Password {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// sessionToken := uuid.NewRandom().String()
	sessionToken := uuid.New().String()

	_, err = cache.Do("SETEX", sessionToken, "300", creds.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(300 * time.Second),
	})

}

func main() {
	initCache()
	handleRequests()
	fmt.Println("hello")
}

func initCache() {
	conn, err := redis.DialURL("redis://localhost")
	if err != nil {
		panic(err)
	}
	cache = conn
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type Member struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `validate:"required,gte=2"` // have to add checks for not been repeated
	Name     string             `validate:"required,gte=2"`
	Lastname string             `validate:"required,gte=2"`
	Mobile   string             `validate:"required,startswith=09,len=11"` // have to add checks for not been repeated
	Email    string             `validate:"required,email"`                // have to add checks for not been repeated
	Password string             `validate:"required,gte=8"`
}

// const tagName = "validate"

// add api validaton
//	send sms when done
//	send email when done
