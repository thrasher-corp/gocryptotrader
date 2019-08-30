package dispatch

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

var mainID uuid.UUID

func TestMain(m *testing.M) {
	var err error
	mainID, err = comms.GetNewID()
	if err != nil {
		log.Fatal(err)
	}

	newChan, err := comms.Subscribe(mainID)
	if err != nil {
		log.Fatal(err)
	}

	go func(c <-chan interface{}) {
		for {
			fmt.Println(<-c)
		}
	}(newChan)

	os.Exit(m.Run())
}

func TestSubscribe(t *testing.T) {
	newChan, err := comms.Subscribe(mainID)
	if err != nil {
		t.Error(err)
	}

	go func(c <-chan interface{}) {
		for {
			fmt.Println(<-c)
		}
	}(newChan)

	_, err = comms.Subscribe(uuid.UUID{})
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestPublish(t *testing.T) {
	nonsensePayload := "NONSENSE!!!!!"
	err := comms.Publish(mainID, nonsensePayload)
	if err != nil {
		t.Error("There was a fully sick error that occured:", err)
	}

	err = comms.Publish(mainID, nil)
	if err == nil {
		t.Error("There was a fully sick error that occured:", err)
	}

	err = comms.Publish(uuid.UUID{}, "invalid uuid")
	if err == nil {
		t.Error("There was a fully sick error that occured:", err)
	}
}

func TestRelease(t *testing.T) {
	err := comms.Release(mainID)
	if err != nil {
		t.Error("OH NOES!!!!:", err)
	}
	err = comms.Release(uuid.UUID{})
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestFullSuite(t *testing.T) {
	var a1, b1, c1, d1, e1 <-chan interface{}
	var a2, b2, c2, d2, e2 <-chan interface{}
	var a3, b3, c3, d3, e3 <-chan interface{}
	specificTicker, err := comms.GetNewID()
	if err != nil {
		t.Fatal(err)
	}

	specificOrderbook, err := comms.GetNewID()
	if err != nil {
		t.Fatal(err)
	}

	randomPayload, err := comms.GetNewID()
	if err != nil {
		t.Fatal(err)
	}

	a1, err = comms.Subscribe(specificTicker)
	if err != nil {
		t.Fatal(err)
	}

	b1, err = comms.Subscribe(specificTicker)
	if err != nil {
		t.Fatal(err)
	}

	c1, err = comms.Subscribe(specificTicker)
	if err != nil {
		t.Fatal(err)
	}

	d1, err = comms.Subscribe(specificTicker)
	if err != nil {
		t.Fatal(err)
	}

	e1, err = comms.Subscribe(specificTicker)
	if err != nil {
		t.Fatal(err)
	}

	a2, err = comms.Subscribe(specificOrderbook)
	if err != nil {
		t.Fatal(err)
	}

	b2, err = comms.Subscribe(specificOrderbook)
	if err != nil {
		t.Fatal(err)
	}

	c2, err = comms.Subscribe(specificOrderbook)
	if err != nil {
		t.Fatal(err)
	}

	d2, err = comms.Subscribe(specificOrderbook)
	if err != nil {
		t.Fatal(err)
	}

	e2, err = comms.Subscribe(specificOrderbook)
	if err != nil {
		t.Fatal(err)
	}

	a3, err = comms.Subscribe(randomPayload)
	if err != nil {
		t.Fatal(err)
	}

	b3, err = comms.Subscribe(randomPayload)
	if err != nil {
		t.Fatal(err)
	}

	c3, err = comms.Subscribe(randomPayload)
	if err != nil {
		t.Fatal(err)
	}

	d3, err = comms.Subscribe(randomPayload)
	if err != nil {
		t.Fatal(err)
	}

	e3, err = comms.Subscribe(randomPayload)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			select {
			case ticker := <-a1:
				fmt.Println("Routine 1 Ticker:", ticker)
			case orderbook := <-a2:
				fmt.Println("Routine 1 Orderbook:", orderbook)
			case random := <-a3:
				fmt.Println("Routine 1 Random:", random)
			}
		}
	}()

	go func() {
		for {
			select {
			case ticker := <-b1:
				fmt.Println("Routine 2 Ticker:", ticker)
			case orderbook := <-b2:
				fmt.Println("Routine 2 Orderbook:", orderbook)
			case random := <-b3:
				fmt.Println("Routine 2 Random:", random)
			}
		}
	}()

	go func() {
		for {
			select {
			case ticker := <-c1:
				fmt.Println("Routine 3 Ticker:", ticker)
			case orderbook := <-c2:
				fmt.Println("Routine 3 Orderbook:", orderbook)
			case random := <-c3:
				fmt.Println("Routine 3 Random:", random)
			}
		}
	}()

	go func() {
		for {
			select {
			case ticker := <-d1:
				fmt.Println("Routine 4 Ticker:", ticker)
			case orderbook := <-d2:
				fmt.Println("Routine 4 Orderbook:", orderbook)
			case random := <-d3:
				fmt.Println("Routine 4 Random:", random)
			}
		}
	}()

	go func() {
		for {
			select {
			case ticker := <-e1:
				fmt.Println("Routine 5 Ticker:", ticker)
			case orderbook := <-e2:
				fmt.Println("Routine 5 Orderbook:", orderbook)
			case random := <-e3:
				fmt.Println("Routine 5 Random:", random)
			}
		}
	}()
	var wgyay sync.WaitGroup
	for i := 0; i < 100; i++ {
		wgyay.Add(1)
		go func() {
			err = comms.Publish(specificTicker, "TICKER PAYLOAD EXAMPLE")
			if err != nil {
				t.Fatal(err)
			}
			wgyay.Done()
		}()

		wgyay.Add(1)
		go func() {
			err = comms.Publish(randomPayload, "Random Payload")
			if err != nil {
				t.Fatal(err)
			}
			wgyay.Done()
		}()

		wgyay.Add(1)
		go func() {
			err = comms.Publish(specificOrderbook, "Random ORDERBOOK")
			if err != nil {
				t.Fatal(err)
			}
			wgyay.Done()
		}()
		time.Sleep(time.Nanosecond * 50)
	}
	wgyay.Wait()

	fmt.Println("REPORT")
	fmt.Println("MAXWORKERS:", MaxWorkers)
	fmt.Println("WORKERS SPAWNED:", comms.count)
}

// type meowCats struct {
// 	wow   []string
// 	rwMtx sync.RWMutex
// 	mtx   sync.Mutex
// }

// func (m *meowCats) RWWriteMeow() {
// 	m.rwMtx.Lock()
// 	m.wow = append(m.wow)
// 	m.rwMtx.Unlock()
// }

// func (m *meowCats) RWDeleteMeow() {
// 	m.rwMtx.Lock()
// 	m.wow = append(m.wow)
// 	m.rwMtx.Unlock()
// }

// func (m *meowCats) RWReadMeow() {
// 	m.rwMtx.RLock()
// 	m.wow = append(m.wow)
// 	m.rwMtx.RUnlock()
// }

// func (m *meowCats) mWriteMeow() {
// 	m.rwMtx.Lock()
// 	m.wow = append(m.wow)
// 	m.rwMtx.Unlock()
// }

// func (m *meowCats) mDeleteMeow() {
// 	m.rwMtx.Lock()
// 	m.wow = append(m.wow)
// 	m.rwMtx.Unlock()
// }

// func (m *meowCats) mReadMeow() {
// 	m.rwMtx.RLock()
// 	m.wow = append(m.wow)
// 	m.rwMtx.RUnlock()
// }

func BenchmarkSubscribe(b *testing.B) {
	// BenchmarkSubscribe-8 3000000	399 ns/op 142 B/op 1 allocs/op
	newID, err := comms.GetNewID()
	if err != nil {
		b.Error(err)
	}

	for n := 0; n < b.N; n++ {
		_, err := comms.Subscribe(newID)
		if err != nil {
			b.Error("MEOW CATS:", err)
		}
	}
}

func BenchmarkPublish(b *testing.B) {
	var idList []uuid.UUID
	// Get 100 ID's
	for x := 0; x < 100; x++ {
		newID, err := comms.GetNewID()
		if err != nil {
			b.Error(err)
		}

		idList = append(idList, newID)

		// Register 100 comms channel per ID's
		for y := 0; y < 5; y++ {
			newChan, err := comms.Subscribe(newID)
			if err != nil {
				b.Error(err)
			}
			// Simulate blocking routine
			go func() {
				for {
					fmt.Println(<-newChan)
				}
			}()
		}
	}

	// Publish change channel to all ID associated comms chanels
	for a := 0; a < b.N; a++ {
		for index := range idList {
			err := comms.Publish(idList[index], "PAYLOAD")
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func TestSomething(t *testing.T) {
	// var channels *chan interface{}

	// var wg sync.WaitGroup
	// wg.Add(5)
	// for i := 0; i < 5; i++ {
	// 	if channels == nil {
	// 		allocation := make(chan interface{})
	// 		channels = &allocation
	// 	} else {
	// 		*channels = make(chan interface{}, len(*channels)+1)
	// 	}
	// 	go func(i int, pComms *chan interface{}, wg *sync.WaitGroup) {
	// 		fmt.Printf("Go Routine %d started \n", i)
	// 		wg.Done()
	// 		fmt.Printf("Go Routine address %v\n", *pComms)
	// 		for {
	// 			fmt.Print("Go Routine Waiting\n\n")
	// 			fmt.Println(<-*pComms, i)
	// 			fmt.Println("Go Routine information received")
	// 		}
	// 	}(i, channels, &wg)
	// }

	// wg.Wait()

	// var state uint32
	// for {
	// 	if atomic.LoadUint32(&state) == 1 {
	// 		break
	// 	}
	// 	select {
	// 	case *channels <- "HELLO":
	// 		fmt.Println("accepting Data")
	// 	default:
	// 		fmt.Println("State change")
	// 		atomic.AddUint32(&state, 1)
	// 	}
	// }

	// time.Sleep(time.Second)
}
