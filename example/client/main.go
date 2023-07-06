package main

import (
	"context"
	"fmt"
	"net/url"
	"ws/ws"
)

func main() {

	//使用方式
	//先获取client实例
	c := ws.NewWebsocket(
		ws.WithClientWsUrl(&url.URL{Scheme: "ws", Host: "127.0.0.1:9501", Path: "ws"}),
	)
	//按照需要设置好监听的闭包函数
	ctx, cancel := context.WithCancel(context.Background())
	//ws连接成功的时候在这里显示
	c.OnOpen(func() {
		fmt.Println("ws 连接成功")
		//发送信息 支持 string []byte{}
		_ = c.ForthwithSend(`你好世界`)
	})
	//ws收到消息的时候在这里显示
	c.OnMessage(func(bytes []byte) {
		fmt.Println("ws 接受到消息: " + string(bytes))
	})
	//ws关闭的时候在这里显示
	c.OnClose(func() {
		fmt.Println("ws 已断开")
		cancel()
	})
	//连接过程中 读取 写入 等操作出现错误或异常在这里显示
	c.OnError(func(err error) {
		fmt.Println("ws 出现错误: " + err.Error())
	})
	c.ForthwithSend("hw")
	//执行连接操作
	if err := c.Connect(); err != nil {
		fmt.Println("ws 连接失败", err.Error())
		return
	}

	//time.Sleep(time.Second * 5)
	//c.Close()
	select {
	case <-ctx.Done():
		break
	}

}
