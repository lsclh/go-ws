package example

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
	"ws/ws"
)

func TestServer(t *testing.T) {
	engine := gin.Default()
	engine.GET("/ws", func(ctx *gin.Context) {

		c := ws.NewWebsocket(
			ws.WithServerHttp(ctx.Writer, ctx.Request),
			ws.WithServerAuth(func(r *http.Request, ctx *ws.Context) bool {
				//读取请求参数进行校验
				//header  r.Header.Get("id")
				//query   r.URL.Query()["id"][0]
				//body	  data := []byte{} r.Body.Read(data)
				//form    r.FormValue()
				ctx.Storage.Set("id", r.URL.Query()["id"][0])
				return true
			}),
		)

		c.OnOpen(func(ctx *ws.Context) {
			fmt.Println("ws 客户端已链接")
		})

		//ws收到消息的时候在这里显示
		c.OnMessage(func(ctx *ws.Context, bytes []byte) {
			fmt.Println("ws 接受到消息: ", ctx.Storage.GetString("id"), string(bytes))
			if string(bytes) == "hw" {
				ctx.ForthwithSend([]byte("你好世界"))
			}
		})
		//ws关闭的时候在这里显示
		c.OnClose(func(ctx *ws.Context) {
			fmt.Println("ws 已断开")
		})
		//连接过程中 读取 写入 等操作出现错误或异常在这里显示
		c.OnError(func(ctx *ws.Context, err error) {
			fmt.Println("ws 出现错误: " + err.Error())
		})
		//执行连接操作
		if err := c.Connect(); err != nil {
			fmt.Println("ws 初始化错误")
		}
	})
	engine.Run(":9501")

}
