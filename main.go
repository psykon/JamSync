package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type client struct {
	source string
	conn   *websocket.Conn
}

type Playlist struct {
	ID     int
	Name   string
	Source string
	Slide  string
}

type Controller struct {
	Playlist []Playlist
	Position int
}

func (c *Controller) Next() {
	if c.Position < len(c.Playlist)-1 {
		c.Position++
	}
}

func (c *Controller) Prev() {
	if c.Position > 0 {
		c.Position--
	}

}
func (c *Controller) Set(i int) {
	c.Position = i
}

func (c *Controller) CurrentSource() string {
	return c.Playlist[c.Position].Source
}
func (c *Controller) CurrentName() string {
	return c.Playlist[c.Position].Name
}
func (c *Controller) CurrentSlide() string {
	return c.Playlist[c.Position].Slide
}

func InitController() {
	readPlaylist()
	C = Controller{Playlist: pl, Position: 0}
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	cpool = []client{}

	pl = []Playlist{}

	C = Controller{}
)

func sendall(message []byte) {
	for _, c := range cpool {
		c.conn.WriteMessage(1, message)
	}
}

func play() {

}

func readPlaylist() {
	content, err := ioutil.ReadFile("./content/playlist.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	var payload []Playlist
	err = json.Unmarshal(content, &payload)
	if err != nil {

		log.Fatal("Error during Unmarshal(): ", err)
	}
	pl = payload

}

func sendCtl() {
	sendall([]byte(fmt.Sprintf("server-source:%s", C.CurrentSource())))
	sendall([]byte(fmt.Sprintf("server-name:%s", C.CurrentName())))
	sendall([]byte(fmt.Sprintf("server-slide:%s", C.CurrentSlide())))
}

func main() {
	InitController()
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Static("/content", "./content")

	// mult := multicast.New(nil, true)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/player", func(c *gin.Context) {
		c.HTML(http.StatusOK, "video.html", gin.H{})
	})
	r.GET("/slides", func(c *gin.Context) {
		c.HTML(http.StatusOK, "slides.html", gin.H{})
	})
	r.GET("/control", func(c *gin.Context) {
		c.HTML(http.StatusOK, "control.html", gin.H{"C": C})
	})

	r.GET("/ws/:source", func(c *gin.Context) {
		s := c.Param("source")

		con, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}

		cl := client{source: s, conn: con}
		cpool = append(cpool, cl)
		// mult.Add(m)
		go func() {
			for {

				mt, message, err := cl.conn.ReadMessage()
				if err != nil {
					log.Println("read:", err, " : ", fmt.Sprint(mt))
				}

				log.Printf("recv: %s : from %s", message, cl.source)
				if string(message) == "control-start" {
					sendCtl()
				} else if string(message) == "control-next" {
					C.Next()
					sendCtl()
				} else if string(message) == "control-prev" {
					C.Prev()
					sendCtl()
				} else if string(message) == "player-ended" {
					C.Next()
					sendCtl()
				} else if strings.HasPrefix(string(message), "control-set:") {
					pos := strings.Split(string(message), ":")
					i, _ := strconv.Atoi(pos[1])
					C.Set(i)
					sendCtl()
				} else {

					sendall(message)
				}
			}
		}()

		// defer cl.conn.Close()
	})

	r.Run() // listen and serve on 0.0.0.0:8080
}
