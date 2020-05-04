package main

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"log"
	"runtime"
)

func rabbitMQRecieve(ch amqp.Channel, Qname string, rabbitMQreturn chan counterRabbitMQ){
	/*conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect rabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		Qname,
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")*/

	msgs, err := ch.Consume(
		Qname,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register consumer")

	var jCounter counterRabbitMQ

	go func() {
		for d := range msgs{
			runtime.Gosched()
			log.Printf("received msgs: %s", d.Body)
			err := json.Unmarshal(d.Body, &jCounter)
			if err != nil {
				panic(err.Error())
			}
			rabbitMQreturn<-jCounter
		}
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	//<-done
}
