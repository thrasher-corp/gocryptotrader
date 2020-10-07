package charts

import (
	"fmt"
	"log"
	"net/http"
)

func (c *Chart) Serve() {
	http.HandleFunc("/", c.handler)
	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Chart) handler(w http.ResponseWriter, r *http.Request) {
	b, err := c.Result()
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
	}
	_,_ = fmt.Fprint(w, b)
}
