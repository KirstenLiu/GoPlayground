package main


import (
	"github.com/gomodule/redigo/redis"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestAPI(t *testing.T) {
	URL := "localhost:8080"
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil{
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(timeHandler)
	handler.ServeHTTP(rr, req)

	if status :=rr.Code; status != http.StatusOK{
		t.Errorf("handler returns wrong status: got %v want %v", status, http.StatusOK)
	}

	expected := "Hi there, local time in vancouver is " + time.Now().Format("2006-01-02 15:04:05") + "!"
	if rr.Body.String() != expected{
		t.Errorf("handler returns wrong body: got\n %v\n want\n %v", rr.Body.String(), expected)
	}
}

func TestCounter(t *testing.T){
	URL := "localhost:8080"
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil{
		t.Fatal(err)
	}

	recorder := make(chan *httptest.ResponseRecorder)
	go hitCountRedisRequest(req, recorder)
	go hitCountRedisRequest(req, recorder)

	conn := pool.Get()
	countInDB, err := redis.Int(conn.Do("GET", "counter"))
	if err != nil{
		panic(err.Error())
	}
	expected := "The hit of this page from db is " + strconv.Itoa(countInDB)
	rr := <- recorder
	if rr.Body.String() != expected{
		t.Errorf("counter returns wrong body: got\n %v\n want\n %v", rr.Body.String(), expected)
	}

	expected2 := "The hit of this page from db is " + strconv.Itoa(countInDB+1)
	rr = <- recorder
	if rr.Body.String() != expected2{
		t.Errorf("counter returns wrong body: got\n %v\n want\n %v", rr.Body.String(), expected)
	}

}

func hitCountRequest(req *http.Request, recorder chan *httptest.ResponseRecorder){
	rr := httptest.NewRecorder()
	handler1 := http.HandlerFunc(hitCountHandler)
	handler1.ServeHTTP(rr, req)

	recorder <- rr
}

func hitCountRedisRequest(req *http.Request, recorder chan *httptest.ResponseRecorder){
	rr := httptest.NewRecorder()
	handler1 := http.HandlerFunc(hitCountRedisHandler)
	handler1.ServeHTTP(rr, req)

	recorder <- rr
}
