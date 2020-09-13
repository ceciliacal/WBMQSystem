package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/lithammer/shortuuid"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Result
type Result struct {
	Id     string      `json:"id"`
	Data   interface{} `json:"data"`
	Topic  string      `json:"topic"`
	Sector string      `json:"sector"`
}

// Ping
type Ping struct {
	CtxStatus string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Bot
type Bot struct {
	Id            string `json:"id"`
	CurrentSector string `json:"current_sector"`
	Topic         string `json:"topic"`
}

// Sensor
type Sensor struct {
	Id            string `json:"id"`
	CurrentSector string `json:"current_sector"`
	Type          string `json:"type"`
}

var bots []Bot
var sensors []Sensor
var warehouses []string
var topics []string
var ch []chan DataEvent
var contextLock = false
var results []Result

func main() {


	router := mux.NewRouter()

	checkCli()
	checkDynamoBotsCache()
	checkDynamoSensorsCache()
	initWarehouseSpaces()
	initTopics()

	//killBot();
	//killRandomSensor();

	router.HandleFunc("/results", getResults).Methods("GET")

	router.HandleFunc("/status", heartBeatMonitoring).Methods("GET")
	router.HandleFunc("/switch", switchContext).Methods("POST")

	// TODO KILL BOT ROUTE
	router.HandleFunc("/bot", killBot).Methods("POST")
	router.HandleFunc("/bot", spawnBot).Methods("POST")
	router.HandleFunc("/bot/{num}", spawnBotRand).Methods("POST")
	router.HandleFunc("/bot", listAllBots).Methods("GET")

	// TODO KILL BOT SENSOR
	router.HandleFunc("/sensor", killSensor).Methods("POST")
	router.HandleFunc("/sensor", spawnSensor).Methods("POST")
	router.HandleFunc("/sensor/{num}", spawnSensorRand).Methods("POST")
	router.HandleFunc("/sensor", listAllSensors).Methods("GET")

	// linea standard per mettere in ascolto l'app. TODO controllo d'errore
	go func() {
		log.Fatal(http.ListenAndServe(":5000", router))
	}()

	// Broker checker ... IDMapToChannel is a bit tricky way to do things ...
	// TODO Need to chek for at-least-one semantic!! (bugs)
	for {
		if len(ch) > 0 {
			for i, c := range ch {
				select {
				case d := <-c:
					results = append(results, Result{
						Id:     IDMapToChannel[i],
						Data:   d.Data,
						Topic:  d.Topic,
						Sector: SectorMapping[i],
					})
					//go printDataEvent(IDMapToChannel[i], d)
				default:
					continue
				}
			}
		}
	}
}



// STATIC OBJECT IN THE SYSTEM
func initTopics() {
	topics = make([]string, 0)
	topics = append(topics,
		"temperature",
		"humidity",
		"motion")
}

// STATIC OBJECT IN THE SYSTEM
func initWarehouseSpaces() {
	warehouses = make([]string, 0)
	warehouses = append(warehouses,
		"a1",
		"a2",
		"a3",
		"a4",
		"b1",
		"b2",
		"c1",
		"c2",
		"c3",
		"c4")
}

func checkCli() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "ctx" {
			contextLock = true
		} else {
			fmt.Println("Wrong argument inserted!")
		}
	}
}

// MAYBE FUTURE IMPLEMENTATION?
func switchContext(w http.ResponseWriter, r *http.Request) {
	if contextLock == true {
		contextLock = false
	} else {
		contextLock = true
	}
}

func heartBeatMonitoring(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var pingNow Ping
	pingNow.Timestamp = time.Now()
	if contextLock == true {
		pingNow.CtxStatus = "alivectx"
		json.NewEncoder(w).Encode(pingNow)
	} else {
		pingNow.CtxStatus = "alive"
		json.NewEncoder(w).Encode(pingNow)
	}
}

func checkDynamoSensorsCache() {
	res, err := GetDBSensors()
	if err != nil {
		panic(err)
	}
	for _, i := range res {
		sensors = append(sensors, i)
		go func() {
			publishTo(i)
		}()
	}
}

func checkDynamoBotsCache() {
	res, err := GetDBBots()
	if err != nil {
		panic(err)
	}
	for _, i := range res {
		bots = append(bots, i)
		chn := make(chan DataEvent)
		ch = append(ch, chn)
		eb.Subscribe(i, chn)
	}
}


func getResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Need to set a ceil value for the results to send ... 50 now
	resultsTrunc := results[:50] // [ [1,2, ... ,50], 51, 52, 53 ... ]
	json.NewEncoder(w).Encode(resultsTrunc)
	results = results[50:] // [50, 51, 52, 53 ... ]
}

func spawnSensorRand(w http.ResponseWriter, r *http.Request) {
	var num = mux.Vars(r)["num"]
	totnum, err := strconv.Atoi(num)
	//json.NewDecoder(r.Body).Decode(&newBot)
	if err != nil {
		// there was an error
		w.WriteHeader(400)
		w.Write([]byte("ID could not be converted to integer"))
		return
	}
	for i := 0; i < totnum; i++ {
		var newSensor Sensor
		newSensor.Id = shortuuid.New()
		newSensor.CurrentSector = warehouses[rand.Intn(len(warehouses))]
		newSensor.Type = topics[rand.Intn(len(topics))]
		sensors = append(sensors, newSensor)
		AddDBSensor(newSensor)
		go func() {
			publishTo(newSensor)
		}()
	}

	// make channel

	// what to return? 200 ? dunno
	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(newSensor)
}

func killRandomSensor() {
	/*
	var num = mux.Vars(r)["num"]
	totnum, err := strconv.Atoi(num)
	//json.NewDecoder(r.Body).Decode(&newBot)
	if err != nil {
		// there was an error
		w.WriteHeader(400)
		w.Write([]byte("ID could not be converted to integer"))
		return
	}
	 */

	for i := 0; i < 2; i++ {

		randomIndex := rand.Intn(len(sensors))
		pickedSensor := sensors[randomIndex]


		var check bool
		check, _ = removeSensor(pickedSensor.Id)
		if check==false {
			fmt.Println("--- Error: sensor was not killed")
		}

	}

	// make channel

	// what to return? 200 ? dunno
	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(newSensor)
}

func killSensor(writer http.ResponseWriter, r *http.Request) {
//func killSensor() {

	var id = "gni5otQjDyhUWmvBE8EWS4"

	var id2 string
	var check bool

	json.NewDecoder(r.Body).Decode(&id2)
	fmt.Println("id sensor: ",id2 )
	//var err bool
	check, _ =removeSensor(id)
	if check==false {
		fmt.Println("--- Error: sensor was not killed")
	}

}



func killBot(writer http.ResponseWriter, r *http.Request) {

	var id = "5JtwUqyifqHu9JoVAPeqJY"
	var id2 string

	var check bool

	json.NewDecoder(r.Body).Decode(&id2)
	fmt.Println("id bot: ",id2 )
	//var err bool
	check, _ =removeBot(id)
	if check==false {
		fmt.Println("--- Error: sensor was not killed")
	}

}

func spawnSensor(w http.ResponseWriter, r *http.Request) {
	var newSensor Sensor
	json.NewDecoder(r.Body).Decode(&newSensor)
	newSensor.Id = shortuuid.New()
	sensors = append(sensors, newSensor)

	// make channel
	AddDBSensor(newSensor)
	go func() {
		publishTo(newSensor)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newSensor)
}

func spawnBotRand(w http.ResponseWriter, r *http.Request) {
	var num = mux.Vars(r)["num"]
	totnum, err := strconv.Atoi(num)
	//json.NewDecoder(r.Body).Decode(&newBot)
	if err != nil {
		// there was an error
		w.WriteHeader(400)
		w.Write([]byte("ID could not be converted to integer"))
		return
	}
	for i := 0; i < totnum; i++ {
		var newBot Bot
		newBot.Id = shortuuid.New()
		newBot.CurrentSector = warehouses[rand.Intn(len(warehouses))]
		newBot.Topic = topics[rand.Intn(len(topics))]
		bots = append(bots, newBot)
		AddDBBot(newBot)
		chn := make(chan DataEvent)
		ch = append(ch, chn)
		eb.Subscribe(newBot, chn)
	}

	// make channel

	// what to return? 200 ? dunno
	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(newBot)
}
//unsubscribes bot with a given Id from current topic
func unsubscribeBot(w http.ResponseWriter, r *http.Request) {

	//var id string
	//var i int
	var id = "2WZtWoLHy5kygkXSegwj56"
	var check bool

	//json.NewDecoder(r.Body).Decode(&id)
	fmt.Println("id unsub: ",id )
	//var err bool
	check, _ =removeTopic(id)
	if check==false {
		fmt.Println("--- Error: topic was not removed")
	}


	var newBot Bot
	newBot, _=GetDBBot(id)
	if newBot.Id!=id {
		fmt.Println("--- Error: wrong bot")
	}

	fmt.Printf("----len", len(eb.subscribersCtx))
	for i := 0; i < len(eb.subscribers); i++ {
		//fmt.Printf("value of a: %d\n", a)
		subscribers := eb.subscribers
		print(&subscribers)
	}

	print(eb.subscribersCtx)

}




func spawnBot(w http.ResponseWriter, r *http.Request) {
	var newBot Bot
	json.NewDecoder(r.Body).Decode(&newBot)
	newBot.Id = shortuuid.New()
	bots = append(bots, newBot)

	fmt.Println("---- Spawning bots!")
	// make channel
	AddDBBot(newBot)
	chn := make(chan DataEvent)
	ch = append(ch, chn)
	eb.Subscribe(newBot, chn)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newBot)
}

func listAllBots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Fillo la response e mando la lista di bots
	json.NewEncoder(w).Encode(bots)
}

func listAllSensors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Fillo la response e mando la lista di bots
	json.NewEncoder(w).Encode(sensors)
}
