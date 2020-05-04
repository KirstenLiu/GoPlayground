package main

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"sync"
	"time"
)
const sendQname = "counterSend"
const receiveQname ="counterReceive"

var mutex = &sync.Mutex{}
var pool = newPool()
var tokenCh = make(chan string)
var dic = make(map[string]chan int)
//var appCh = make(chan counterRabbitMQ)
var counter int

type counterRabbitMQ struct{
	Token string
	Count int
}



func main() {
	http.HandleFunc("/", timeHandler)
	//log.Fatal(http.ListenAndServe(":8080", nil))
	//done := make(chan bool, 1)
	//go handlerChannel(done)
	//<- done

	//http.HandleFunc("/hitcount", hitCountHandler)
	//log.Fatal(http.ListenAndServe(":8080", nil))

	http.HandleFunc("/hitcountRedis", jsonHitCountHandler)
	//json approach in redis
	conn := pool.Get()
	json, err := json.Marshal(0)
	if err != nil {
		panic(err.Error())
	}
	_, err = conn.Do("SET", "json:counter", json)
	if err !=nil{
		panic(err.Error())
	}

	//http.HandleFunc("/jsonHitCount", jsonHitCountHandler)
	//initial counter to 0
	//counter = 0

	go jsonRabbitMQHitCount()
	go counterApp()

	http.HandleFunc("/rabbitMQcount", hitCountRabbitMQHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func timeHandler(w http.ResponseWriter, r *http.Request) {
	//w.WriteHeader(http.StatusOK)
	//w.Header().Set("Content-Type", "application/json")
	//fmt.Fprintf(w, "Hi there, local time in vancouver is %s!", time.Now().Format("2006-01-02 15:04:05"))
	//io.WriteString(w, `{"alive": true}`)

	done := make(chan bool, 1)
	go setHeaderOK(w, done)
	<- done

	msg := make(chan string)
	go printTime(msg)
	fmt.Fprint(w, <-msg)

}

func hitCountHandler(w http.ResponseWriter, r *http.Request){
	//mutex.Lock()

	hit := make(chan int)
	go hitCount(hit)
	localCount := <-hit

	fmt.Fprint(w, "The hit of this page is ", localCount)
	//mutex.Unlock()
}

func hitCountRedisHandler(w http.ResponseWriter, r *http.Request){
	hit := make(chan int)
	go hitCountRedis(hit)
	localCount := <-hit

	fmt.Fprint(w, "The hit of this page from db is ", localCount)
}

func hitCountRabbitMQHandler(w http.ResponseWriter, r *http.Request){
	//create token
	token := generateToken()

	//create channel for goroutine to return
	hit := make(chan int)
	//NEED: close the channel you don't need
	defer close(hit)

	mutex.Lock()
	//write to dic <token, return channel>
	dic[token] = hit
	mutex.Unlock()
	log.Printf("ch created in handler: %v", hit)

	//pass token to goroutine listening in main
	tokenCh <- token
	log.Printf("send token %v to channel", token)

	MQcounter := <- hit
	log.Printf("get counter %v back from ch", MQcounter)
	fmt.Fprint(w, "The hit of this page from rabbitMQ is ", MQcounter)
	//NEED: delete dic that returned already

	mutex.Lock()
	delete(dic, token)
	mutex.Unlock()
}

func printTime(msg chan string){
	time :=  "Hi there, local time in vancouver is " + time.Now().Format("2006-01-02 15:04:05") + "!"
	msg <- time
}

func setHeaderOK(w http.ResponseWriter, done chan bool){
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	done <- true
}

func hitCount(hit chan int){
	mutex.Lock()
	counter ++
	myCount := counter
	mutex.Unlock()

	hit <- myCount
}

func hitCountRedis(hit chan int){
	conn := pool.Get()
	defer conn.Close()


	//mutex.Lock()
	counterRedis, err := redis.Int(conn.Do("INCR", "counter"))
	if err !=nil{
		panic(err.Error())
	}
	//counter = counterRedis
	//myCount := counterRedis
	//mutex.Unlock()

	hit <- counterRedis
}

func jsonHitCountHandler(w http.ResponseWriter, r *http.Request){

	hit := make(chan int)
	go jsonHitCount(hit)
	localCount := <-hit

	fmt.Fprint(w, "The hit json of this page is ", localCount)
}

func jsonHitCount(hit chan int){
	conn := pool.Get()
	defer conn.Close()

	counterJson, err := redis.Int(conn.Do("GET", "json:counter"))
	if err != nil {
		panic(err.Error())
	}

	mutex.Lock()
	counter = counterJson + 1
	myCount := counter
	mutex.Unlock()

	json, err := json.Marshal(counter)
	if err != nil {
		panic(err.Error())
	}

	mutex.Lock()
	_, err = conn.Do("SET", "json:counter", json)
	if err !=nil{
		panic(err.Error())
	}
	mutex.Unlock()

	hit <- myCount
}

func jsonRabbitMQHitCount(){

	connR, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect rabbitMQ")
	defer connR.Close()

	ch, err := connR.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	_, err = ch.QueueDeclare(
		sendQname,
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	_, err = ch.QueueDeclare(
		receiveQname,
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	appCh := make(chan counterRabbitMQ)

	for {
		//get token by global channel
		token := <-tokenCh
		/*if close != true{
			panic(close)
		}*/
		log.Printf("get token %v by channel", token)

		//send token to rabbitMQ(send)
		data := counterRabbitMQ{
			Token: token,
			Count: 0,
		}
		var jsonData []byte
		jsonData, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}
		log.Printf("sending json %v", string(jsonData))


		//send to rabbitMQ
		err = ch.Publish(
			"",     // exchange
			sendQname, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing {
				ContentType: "text/plain",
				Body:        jsonData,
			})
		failOnError(err, "Failed to publish a message")

		//call counterApp (receive)
		//counterApp(ch,  counterCh)

		//dequeue rabbitMQ, get counter
		rabbitMQRecieve(*ch,receiveQname, appCh)
		myCount := <-appCh
		//close(counterCh)
		log.Printf("recieve from rabbitMQ %v", myCount)

		//counterR = myCount.Count

		//use token to find channel to return
		//NEED: if write is not thread-safe, read is not thread-safe either
		mutex.Lock()
		ch := dic[myCount.Token]
		//log.Println(dic)
		//log.Println(ch)
		mutex.Unlock()
		//send counter back to specific channel
		ch <- myCount.Count
	}
}

func counterApp(){

	counter := 0

	connR, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect rabbitMQ")
	defer connR.Close()

	ch, err := connR.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	_, err = ch.QueueDeclare(
		sendQname,
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	_, err = ch.QueueDeclare(
		receiveQname,
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	counterCh := make(chan counterRabbitMQ)

	for {
		////Counter stay local in counterApp now so this
		//dequeue rabbitMQ
		rabbitMQRecieve(*ch, sendQname, counterCh)
		countMQ := <-counterCh
		log.Printf("in app, recieve token and counter %v", countMQ)
		counter = counter + 1
		countMQ.Count = counter
		//marshall counter
		json, err := json.Marshal(countMQ)
		if err != nil {
			panic(err)
		}
		//send respond to rabbitMQ
		err = ch.Publish(
			"",        // exchange
			receiveQname, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        json,
			})
		failOnError(err, "Failed to publish a message")
		log.Printf("in app, send %v to rabbitMQ", countMQ)
	}
}

func generateToken() string{
	b := uuid.New()
	return  b.String()
}

