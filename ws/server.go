package ws

import (
	"github.com/gorilla/websocket"
	"net/http"
)

/*
	使用方式
	engine := gin.Default()
	engine.GET("/ws", func(ctx *gin.Context) {
		c, err := ws.NewWebsocketServer(&ws.Config{
			Timeout: time.Second * 20,
		}, ctx.Writer, ctx.Request)
		if err != nil {
			return
		}

		//ws收到消息的时候在这里显示
		c.OnMessage(func(bytes []byte) {
			fmt.Println("ws 接受到消息: " + string(bytes))
		})
		//ws关闭的时候在这里显示
		c.OnClose(func() {
			fmt.Println("ws 已断开")
		})
		//连接过程中 读取 写入 等操作出现错误或异常在这里显示
		c.OnError(func(err error) {
			fmt.Println("ws 出现错误: " + err.Error())
		})
	})
	engine.Run(":9501")
*/

func NewWebsocketServer(conf *Config, w http.ResponseWriter, r *http.Request) (cl *client, err error) {
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(w, r, nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if conf.KeepAliveMessage == "" {
		conf.KeepAliveMessage = "ping"
	}
	cl = &client{
		keepAlive:        conf.KeepAlive,
		timeout:          conf.Timeout,
		keepAliveMessage: []byte(conf.KeepAliveMessage),
		conn:             conn,
		isClose:          false,
		connHost:         nil,   //服务端不存在链接地址
		isReconnect:      false, //不需要处理重连
		serviceType:      SERVER,
	}
	cl.ws()
	return
}