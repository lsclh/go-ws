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
		c := ws.NewWebsocket(
			ws.WithServerHttp(ctx.Writer, ctx.Request),
		)

		c.OnOpen(func() {
			fmt.Println("ws 客户端已链接")
			time.Sleep(time.Second * 5)
			c.Close()
		})
		//ws收到消息的时候在这里显示
		c.OnMessage(func(bytes []byte) {
			fmt.Println("ws 接受到消息: " + string(bytes))
			if string(bytes) == "hw" {
				c.ForthwithSend("你好世界")
			}
		})
		//ws关闭的时候在这里显示
		c.OnClose(func() {
			fmt.Println("ws 已断开")
		})
		//连接过程中 读取 写入 等操作出现错误或异常在这里显示
		c.OnError(func(err error) {
			fmt.Println("ws 出现错误: " + err.Error())
		})
		//执行连接操作
		if err := c.Connect(); err != nil {
			fmt.Println("ws 初始化错误")
		}
	})
	engine.Run(":9501")

}
