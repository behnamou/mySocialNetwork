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
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kavenegar/kavenegar-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-redis/redis/v8"
)

//var cache redis.Conn

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

func main() {
	//initCache()
	handleRequests()
	fmt.Println("hello")
}

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

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/v1/auth/register", homepage).Methods(http.MethodPost)

	myRouter.HandleFunc("/login", SignIn).Methods(http.MethodPost)

	myRouter.HandleFunc("/test", TestLoggedIn).Methods(http.MethodGet)

	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var loggedInMember Member

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

	if err = memberCollection.FindOne(ctx, bson.M{
		"username": creds.Username,
	}).Decode(&loggedInMember); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(loggedInMember.Password), []byte(creds.Password))
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// sessionToken := uuid.NewRandom().String()
	sessionToken := uuid.New().String()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	err = redisClient.Set(ctx, sessionToken, creds.Username, 300*time.Second).Err()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//_, err = cache.Do("SETEX", sessionToken, "300", creds.Username)
	//if err != nil {
	//	w.WriteHeader(http.StatusInternalServerError)
	//	return
	//}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(300 * time.Second),
	})

}

//func initCache() {
//conn, err := redis.DialURL("redis://localhost")
//if err != nil {
//	panic(err)
//}
//cache = conn
//}

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

func TestLoggedIn(w http.ResponseWriter, r *http.Request) {
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//sessionToken := c.Value

	//response := redis.Conn{}.Get(ctx, sessionToken)

	//response, err := cache.Do("GET", sessionToken)
	//if err != nil {
	//	w.WriteHeader(http.StatusInternalServerError)
	//	return
	//}

	//if response == nil {
	//	w.WriteHeader(http.StatusUnauthorized)
	//	return
	//}

	w.Write([]byte(fmt.Sprintf("Test Passed\nWelcome %s!", c)))

}
