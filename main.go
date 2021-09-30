package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
	"ws/ws"
)

func main() {
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

}
