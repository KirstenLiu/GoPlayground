package main

import (
	"log"
)

func failOnError(err error, msg string){
	if err != nil{
		log.Fatalf("%s: %s", msg, err)
	}
}

/*func rabbitMQSend(Qname string, body []byte){
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "fail to connect to rabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "fail to open channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		Qname, // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	//body = "Hello World!"
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing {
			ContentType: "text/plain",
			Body:        body,
		})
	log.Printf(" [x] Sent %v", body)
	failOnError(err, "Failed to publish a message")
}*/
